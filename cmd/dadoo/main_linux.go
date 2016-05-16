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
	"github.com/kr/pty"
	"github.com/opencontainers/runc/libcontainer/system"
)

func main() {
	var logFile string
	flag.StringVar(&logFile, "log", "dadoo.log", "dadoo log file path")

	var (
		stdoutPath, stdinPath, stderrPath string
		isTTY                             bool
	)
	flag.StringVar(&stdoutPath, "stdout", "", "path to stdout")
	flag.StringVar(&stdinPath, "stdin", "", "path to stdin")
	flag.StringVar(&stderrPath, "stderr", "", "path to stderr")
	flag.BoolVar(&isTTY, "tty", false, "create a TTY")

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

	signals := make(chan os.Signal, 100)
	signal.Notify(signals, syscall.SIGCHLD)

	pidFilePath := filepath.Join(bundlePath, "pidfile")

	// we need to be the subreaper so we can wait on the detached container process
	system.SetSubreaper(os.Getpid())

	var runcCmd *exec.Cmd
	if command == "run" {
		// listen to an exit socket early so waiters can wait for dadoo
		dadoo.Listen(filepath.Join(bundlePath, "exit.sock"))
		runcCmd = exec.Command(runtime, "-debug", "-log", logFile, "start")
	}

	if command == "exec" {
		runcCmd = exec.Command(runtime, "-debug", "-log", logFile, "exec")
	}

	runcArgs := []string{"-d", "-pid-file", pidFilePath}
	runcCmd.Dir = bundlePath

	if stdinPath != "" {
		stdin, err := os.Open(stdinPath)
		check(err)
		runcCmd.Stdin = stdin
	}

	if stdoutPath != "" {
		stdout, err := os.OpenFile(stdoutPath, os.O_WRONLY|os.O_APPEND, 0600)
		check(err)
		runcCmd.Stdout = stdout
	}

	if stderrPath != "" {
		stderr, err := os.OpenFile(stderrPath, os.O_WRONLY|os.O_APPEND, 0600)
		check(err)
		runcCmd.Stderr = stderr
	}

	if isTTY {
		p, t, _ := pty.Open()
		runcArgs = append(runcArgs, "-tty", "-console", t.Name())

		if runcCmd.Stdout != nil {
			go io.Copy(runcCmd.Stdout, p)
			runcCmd.Stdout = nil
		}
		if runcCmd.Stdin != nil {
			go io.Copy(p, runcCmd.Stdin)
			runcCmd.Stdin = nil
		}
	}

	var processJSONPath string
	if command == "exec" {
		processJSONContents, _ := ioutil.ReadAll(os.Stdin)
		f, _ := ioutil.TempFile(bundlePath, "process-")
		f.Write(processJSONContents)
		f.Close()
		processJSONPath = f.Name()

		runcArgs = append(runcArgs, "-process", processJSONPath)
	}

	runcCmd.Args = append(runcCmd.Args, append(runcArgs, containerId)...)
	if err := runcCmd.Start(); err != nil {
		fd3.Write([]byte{2})
		os.Exit(2)
	}

	containerPid := -2
	for range signals {
		exits := make(map[int]int)
		for {
			var status syscall.WaitStatus
			var rusage syscall.Rusage
			wpid, err := syscall.Wait4(-1, &status, syscall.WNOHANG, &rusage)

			if err != nil || wpid <= 0 {
				break // wait for next SIGCHLD
			}

			if wpid == runcCmd.Process.Pid {
				fd3.Write([]byte{byte(status.ExitStatus())})

				if status.ExitStatus() != 0 {
					os.Exit(3) // nothing to wait for, container didn't launch
				}

				if processJSONPath != "" {
					os.Remove(processJSONPath)
				}

				containerPid, err = readPid(pidFilePath)
				check(err)
			}

			if wpid == containerPid || containerPid < 0 {
				exits[wpid] = status.ExitStatus()
			}

			if status, ok := exits[containerPid]; ok {
				if command == "run" {
					check(exec.Command(runtime, "delete", containerId).Run())
				}

				os.Exit(status)
			}
		}
	}
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
