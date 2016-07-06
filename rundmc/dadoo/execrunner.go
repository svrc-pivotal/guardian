package dadoo

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/cloudfoundry-incubator/garden"
	"github.com/cloudfoundry-incubator/guardian/rundmc/runrunc"
	"github.com/cloudfoundry/gunk/command_runner"
	"github.com/kr/logfmt"
	"github.com/pivotal-golang/lager"
)

//go:generate counterfeiter . PidGetter
type PidGetter interface {
	Pid(pidFilePath string) (int, error)
}

type ExecRunner struct {
	dadooPath      string
	runcPath       string
	processIDGen   runrunc.UidGenerator
	pidGetter      PidGetter
	iodaemonRunner runrunc.ExecRunner
	commandRunner  command_runner.CommandRunner
}

func NewExecRunner(dadooPath, runcPath string, processIDGen runrunc.UidGenerator, pidGetter PidGetter, iodaemonRunner runrunc.ExecRunner, commandRunner command_runner.CommandRunner) *ExecRunner {
	return &ExecRunner{
		dadooPath:      dadooPath,
		runcPath:       runcPath,
		processIDGen:   processIDGen,
		pidGetter:      pidGetter,
		iodaemonRunner: iodaemonRunner,
		commandRunner:  commandRunner,
	}
}

func (d *ExecRunner) Run(log lager.Logger, spec *runrunc.PreparedSpec, processesPath, handle string, tty *garden.TTYSpec, pio garden.ProcessIO) (p garden.Process, theErr error) {
	if !contains(spec.Env, "USE_DADOO=true") {
		return d.iodaemonRunner.Run(log, spec, processesPath, handle, tty, pio)
	}

	log = log.Session("execrunner")

	log.Info("start")
	defer log.Info("done")

	processID := d.processIDGen.Generate()

	processPath := filepath.Join(processesPath, processID)
	if err := os.MkdirAll(processPath, 0700); err != nil {
		return nil, err
	}

	fd3r, fd3w, err := os.Pipe()
	if err != nil {
		return nil, err
	}

	logr, logw, err := os.Pipe()
	if err != nil {
		return nil, err
	}

	defer fd3r.Close()
	defer logr.Close()

	pipes, err := mkFifos(processPath)
	if err != nil {
		return nil, err
	}

	var cmd *exec.Cmd
	if tty != nil {
		cmd = exec.Command(d.dadooPath, "-tty", "-uid", strconv.Itoa(spec.HostUID), "-gid", strconv.Itoa(spec.HostGID), "exec", d.runcPath, processPath, handle)
	} else {
		cmd = exec.Command(d.dadooPath, "exec", d.runcPath, processPath, handle)
	}

	cmd.ExtraFiles = []*os.File{
		fd3w,
		logw,
	}

	encodedSpec, err := json.Marshal(spec.Process)
	if err != nil {
		return nil, err // this could *almost* be a panic: a valid spec should always encode (but out of caution we'll error)
	}

	cmd.Stdin = bytes.NewReader(encodedSpec)
	if err := d.commandRunner.Start(cmd); err != nil {
		return nil, err
	}

	go d.commandRunner.Wait(cmd) // wait on spawned process to avoid zombies

	fd3w.Close()
	logw.Close()

	log.Info("open-pipes")

	if err := pipes.start(pio, TtySize{Rows: uint16(tty.WindowSize.Rows), Cols: uint16(tty.WindowSize.Columns)}); err != nil {
		return nil, err
	}

	log.Info("read-exit-fd")

	runcExitStatus := make([]byte, 1)
	fd3r.Read(runcExitStatus)

	log.Info("runc-exit-status", lager.Data{"status": runcExitStatus[0]})

	defer func() {
		theErr = processLogs(log, logr, theErr)
	}()

	if runcExitStatus[0] != 0 {
		return nil, fmt.Errorf("exit status %d", runcExitStatus[0])
	}

	return d.newProcess(pipes, filepath.Join(processPath, "pidfile")), nil
}

func (d *ExecRunner) Attach(log lager.Logger, processID string, io garden.ProcessIO, processesPath string) (garden.Process, error) {
	return d.iodaemonRunner.Attach(log, processID, io, processesPath)
}

type osSignal garden.Signal

func (s osSignal) OsSignal() syscall.Signal {
	switch garden.Signal(s) {
	case garden.SignalTerminate:
		return syscall.SIGTERM
	default:
		return syscall.SIGKILL
	}
}

type process struct {
	pidFilePath string
	pidGetter   PidGetter
	pipes       *fifos
}

func (d *ExecRunner) newProcess(pipes *fifos, pidFilePath string) *process {
	return &process{
		pipes:       pipes,
		pidFilePath: pidFilePath,
		pidGetter:   d.pidGetter,
	}
}

func (p *process) ID() string {
	return ""
}

func (p *process) Wait() (int, error) {
	return p.pipes.wait()
}

func (p *process) SetTTY(tty garden.TTYSpec) error {
	p.pipes.send(TtySize{
		Rows: uint16(tty.WindowSize.Rows),
		Cols: uint16(tty.WindowSize.Columns),
	})

	return nil
}

func (p *process) Signal(signal garden.Signal) error {
	pid, err := p.pidGetter.Pid(p.pidFilePath)
	if err != nil {
		return errors.New(fmt.Sprintf("fetching-pid: %s", err))
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return errors.New(fmt.Sprintf("finding-process: %s", err))
	}

	return process.Signal(osSignal(signal).OsSignal())
}

type fifos struct {
	stdin, stdout, stderr, exit, winsz, exitcode string
	exitted                                      chan struct{}
	winszCh                                      chan TtySize
}

func mkFifos(dir string) (*fifos, error) {
	stdin, stdout, stderr, winsz, exit, exitcode := filepath.Join(dir, "stdin"),
		filepath.Join(dir, "stdout"),
		filepath.Join(dir, "stderr"),
		filepath.Join(dir, "winsz"),
		filepath.Join(dir, "exit"),
		filepath.Join(dir, "exitcode")

	for _, pipe := range []string{stdin, stdout, stderr, winsz, exit} {
		if err := syscall.Mkfifo(pipe, 0); err != nil {
			return nil, err
		}
	}

	return &fifos{
		stdin:    stdin,
		stdout:   stdout,
		stderr:   stderr,
		winsz:    winsz,
		exit:     exit,
		exitcode: exitcode,
		exitted:  make(chan struct{}),
		winszCh:  make(chan TtySize),
	}, nil
}

func (f fifos) start(pio garden.ProcessIO, ttySize TtySize) error {
	fmt.Println("################################## 1")
	stdin, err := os.OpenFile(f.stdin, os.O_WRONLY, 0600)
	if err != nil {
		return err
	}

	if pio.Stdin != nil {
		go io.Copy(stdin, pio.Stdin)
	}

	fmt.Println("################################## 2")
	stdout, err := os.Open(f.stdout)
	if err != nil {
		return err
	}

	if pio.Stdout != nil {
		go io.Copy(pio.Stdout, stdout)
	}

	fmt.Println("################################## 3")
	stderr, err := os.Open(f.stderr)
	if err != nil {
		return err
	}

	if pio.Stderr != nil {
		go io.Copy(pio.Stderr, stderr)
	}

	fmt.Println("################################## 4")
	winSize, err := os.OpenFile(f.winsz, os.O_WRONLY, 0600)
	if err != nil {
		return err
	}

	if err := json.NewEncoder(winSize).Encode(ttySize); err != nil {
		return err
	}

	go func() {
		for {
			select {
			case s := <-f.winszCh:
				json.NewEncoder(winSize).Encode(s)
			case <-f.exitted:
				return
			}
		}
	}()

	exit, err := os.Open(f.exit)
	if err != nil {
		return err
	}

	go func() {
		buf := make([]byte, 1)
		exit.Read(buf)
		close(f.exitted)
	}()

	return nil
}

func (f fifos) wait() (int, error) {
	<-f.exitted

	if _, err := os.Stat(f.exitcode); os.IsNotExist(err) {
		return 1, fmt.Errorf("could not find the exitcode file for the process: %s", err.Error())
	}

	exitcode, err := ioutil.ReadFile(f.exitcode)
	if err != nil {
		return 1, err
	}

	if len(exitcode) == 0 {
		return 1, fmt.Errorf("the exitcode file is empty")
	}

	code, err := strconv.Atoi(string(exitcode))
	if err != nil {
		return 1, fmt.Errorf("failed to parse exit code: %s", err.Error())
	}

	return code, nil
}

func (f fifos) send(s TtySize) {
	f.winszCh <- s
}

func contains(envVars []string, envVar string) bool {
	for _, e := range envVars {
		if e == envVar {
			return true
		}
	}
	return false
}

func processLogs(log lager.Logger, logs io.Reader, err error) error {
	buff, readErr := ioutil.ReadAll(logs)
	if readErr != nil {
		return fmt.Errorf("start: read log file: %s", readErr)
	}

	forwardRuncLogsToLager(log, buff)

	if err != nil {
		return wrapWithErrorFromRuncLog(log, err, buff)
	}

	return nil
}

func forwardRuncLogsToLager(log lager.Logger, buff []byte) {
	parsedLogLine := struct{ Msg string }{}
	for _, logLine := range strings.Split(string(buff), "\n") {
		if err := logfmt.Unmarshal([]byte(logLine), &parsedLogLine); err == nil {
			log.Debug("runc", lager.Data{
				"message": parsedLogLine.Msg,
			})
		}
	}
}

func wrapWithErrorFromRuncLog(log lager.Logger, originalError error, buff []byte) error {
	parsedLogLine := struct{ Msg string }{}
	logfmt.Unmarshal(buff, &parsedLogLine)
	return fmt.Errorf("runc exec: %s: %s", originalError, parsedLogLine.Msg)
}
