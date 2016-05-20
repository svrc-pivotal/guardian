package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/cloudfoundry-incubator/guardian/rundmc/dadoo"
	"github.com/opencontainers/runc/libcontainer/system"
)

func main() {
	fmt.Println("dadoo is here")
	var logFile string
	flag.StringVar(&logFile, "log", "dadoo.log", "dadoo log file path")

	var (
		stdoutPath, stdinPath, stderrPath string
		processJSONPath                   string
	)
	flag.StringVar(&stdoutPath, "stdout", "", "path to stdout")
	flag.StringVar(&stdinPath, "stdin", "", "path to stdin")
	flag.StringVar(&stderrPath, "stderr", "", "path to stderr")
	flag.StringVar(&processJSONPath, "process", "", "path to the process.json file")

	flag.Parse()

	command := flag.Args()[0] // e.g. run
	runtime := flag.Args()[1] // e.g. runc
	bundlePath := flag.Args()[2]
	containerId := flag.Args()[3]

	if command != "run" && command != "exec" {
		fmt.Fprintf(os.Stderr, "unknown command: %s", command)
		os.Exit(127)
	}

	fd3 := os.NewFile(3, "/proc/self/fd/3")

	r := reaper{}
	r.init()

	pidFilePath := filepath.Join(bundlePath, "pidfile")

	// we need to be the subreaper so we can wait on the detached container process
	system.SetSubreaper(os.Getpid())

	var runcCmd *exec.Cmd
	if command == "run" {
		// listen to an exit socket early so waiters can wait for dadoo
		dadoo.Listen(filepath.Join(bundlePath, "exit.sock"))
		runcCmd = startCommand(runtime, bundlePath, containerId, logFile)
		runcCmd.Stdout = os.Stdout
		runcCmd.Stderr = os.Stderr
	}

	if command == "exec" {
		runcCmd = execCommand(runtime, bundlePath, containerId, logFile, stdinPath, stdoutPath, stderrPath, pidFilePath, processJSONPath)
		runcCmd.Stdout = io.MultiWriter(os.Stdout, runcCmd.Stdout)
		runcCmd.Stderr = io.MultiWriter(os.Stderr, runcCmd.Stderr)
	}

	if err := runcCmd.Start(); err != nil {
		fd3.Write([]byte{2})
		os.Exit(2)
	}

	r.onExit(runcCmd.Process.Pid, func(status syscall.WaitStatus) {
		fd3.Write([]byte{byte(status.ExitStatus())})
		if status.ExitStatus() != 0 {
			os.Exit(3) // nothing to wait for, container didn't launch
		}

		containerPid, err := readPid(pidFilePath)
		check(err)

		r.onExit(containerPid, func(status syscall.WaitStatus) {
			check(exec.Command(runtime, "delete", containerId).Run())
			os.Exit(status.ExitStatus())
		})
	})

	r.reap()
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func readPid(pidFile string) (int, error) {
	b, err := ioutil.ReadFile(pidFile)
	if err != nil {
		return -1, err
	}

	var pid int
	if _, err := fmt.Sscanf(string(b), "%d", &pid); err != nil {
		return -1, err
	}

	return pid, nil
}

func forwardReadFIFO(path string) io.Reader {
	r, err := os.Open(path)
	check(err)

	return r
}

func forwardWriteFIFO(path string) io.Writer {
	w, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0600)
	check(err)

	return w
}

func startCommand(runtime, bundlePath, containerId, logFile string) *exec.Cmd {
	runcCmd := exec.Command(runtime, "-debug", "-log", logFile, "start", containerId)
	runcCmd.Dir = bundlePath

	return runcCmd
}

func execCommand(runtime, bundlePath, containerId, logFile, stdinPath, stdoutPath, stderrPath, pidFilePath, processJSONPath string) *exec.Cmd {
	runcCmd := exec.Command(
		runtime,
		"-debug", "-log", logFile,
		"exec", "-d", "-pid-file", pidFilePath, "-process", processJSONPath,
		containerId,
	)
	runcCmd.Dir = bundlePath
	runcCmd.Stdin = forwardReadFIFO(stdinPath)
	runcCmd.Stdout = forwardWriteFIFO(stdoutPath)
	runcCmd.Stderr = forwardWriteFIFO(stderrPath)

	return runcCmd
}

type reaper struct {
	signals       chan os.Signal
	exitCallbacks map[int]func(syscall.WaitStatus)
	exittedProcs  map[int]syscall.WaitStatus
}

func (r *reaper) init() {
	r.signals = make(chan os.Signal, 100)
	signal.Notify(r.signals, syscall.SIGCHLD)

	r.exitCallbacks = make(map[int]func(syscall.WaitStatus))
	r.exittedProcs = make(map[int]syscall.WaitStatus)
}

func (r *reaper) reap() {
	for range r.signals {
		for {
			var status syscall.WaitStatus
			var rusage syscall.Rusage
			wpid, err := syscall.Wait4(-1, &status, syscall.WNOHANG, &rusage)
			if err != nil || wpid <= 0 {
				break // wait for next SIGCHLD
			}

			r.exitted(wpid, status)
		}
	}
}

func (r *reaper) exitted(pid int, status syscall.WaitStatus) {
	cb, ok := r.exitCallbacks[pid]
	if ok {
		cb(status)
		delete(r.exitCallbacks, pid)
	} else {
		r.exittedProcs[pid] = status
	}
}

func (r *reaper) onExit(pid int, cb func(syscall.WaitStatus)) {
	status, ok := r.exittedProcs[pid]
	if ok {
		cb(status)
		delete(r.exittedProcs, pid)
	} else {
		r.exitCallbacks[pid] = cb
	}
}

// func a(signals chan os.Signal) {
// 	containerPid := -2
// 	for range signals {
// 		exits := make(map[int]int)
// 		for {
// 			var status syscall.WaitStatus
// 			var rusage syscall.Rusage
// 			wpid, err := syscall.Wait4(-1, &status, syscall.WNOHANG, &rusage)
// 			if err != nil || wpid <= 0 {
// 				break // wait for next SIGCHLD
// 			}

// 			if wpid == runcCmd.Process.Pid {
// 				fd3.Write([]byte{byte(status.ExitStatus())})

// 				if status.ExitStatus() != 0 {
// 					os.Exit(3) // nothing to wait for, container didn't launch
// 				}

// 				containerPid, err = readPid(pidFilePath)
// 				check(err)
// 			}

// 			if wpid == containerPid || containerPid < 0 {
// 				exits[wpid] = status.ExitStatus()
// 			}

// 			if status, ok := exits[containerPid]; ok {
// 				check(exec.Command(runtime, "delete", containerId).Run())
// 				os.Exit(status)
// 			}
// 		}
// 	}
// }
