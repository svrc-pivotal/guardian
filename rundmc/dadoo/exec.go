package dadoo

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/cloudfoundry-incubator/garden"
	"github.com/cloudfoundry-incubator/guardian/rundmc/runrunc"
	"github.com/cloudfoundry/gunk/command_runner"
	"github.com/pivotal-golang/lager"
)

//go:generate counterfeiter . ExitWaiter
type ExitWaiter interface {
	Wait(path string) (<-chan struct{}, error)
}

type DadooExecer struct {
	dadooPath    string
	runner       command_runner.CommandRunner
	pidGenerator runrunc.UidGenerator
	exitWaiter   ExitWaiter
}

func NewExecer(dadooPath string, runner command_runner.CommandRunner, pidGenerator runrunc.UidGenerator, exitWaiter ExitWaiter) *DadooExecer {
	return &DadooExecer{
		runner:       runner,
		pidGenerator: pidGenerator,
		dadooPath:    dadooPath,
		exitWaiter:   exitWaiter,
	}
}

func (d *DadooExecer) Exec(log lager.Logger, bundlePath, handle string, spec garden.ProcessSpec, processIO garden.ProcessIO) (garden.Process, error) {
	processId := d.pidGenerator.Generate()

	if err := os.Mkdir(filepath.Join(bundlePath, "processes"), 0700); err != nil {
		panic(err)
	}

	stdinPipe := filepath.Join(bundlePath, "processes", fmt.Sprintf("%s.stdin", processId))
	if err := syscall.Mkfifo(stdinPipe, 0); err != nil {
		panic(err)
	}

	stdoutPipe := filepath.Join(bundlePath, "processes", fmt.Sprintf("%s.stdout", processId))
	if err := syscall.Mkfifo(stdoutPipe, 0); err != nil {
		panic(err)
	}

	stderrPipe := filepath.Join(bundlePath, "processes", fmt.Sprintf("%s.stderr", processId))
	if err := syscall.Mkfifo(stderrPipe, 0); err != nil {
		panic(err)
	}

	waitSock := filepath.Join(bundlePath, "processes", fmt.Sprintf("%s.sock", processId))

	cmd := exec.Command(d.dadooPath, "-stdin", stdinPipe, "-stdout", stdoutPipe, "-stderr", stderrPipe, "-waitSock", waitSock, "exec", "runc", bundlePath, handle)
	cmd.Stdout = os.Stdout

	process := NewProcess(d.exitWaiter, processId, waitSock)

	bytes, _ := json.Marshal(spec)
	processJSON := strings.NewReader(string(bytes))
	cmd.Stdin = processJSON

	if err := d.runner.Start(cmd); err != nil {
		panic(err)
	}

	fmt.Println("openning stdin pipe")
	stdinP, err := os.OpenFile(stdinPipe, os.O_WRONLY, 0600)
	if err != nil {
		panic(err)
	}
	fmt.Println("stdin pipe is open")

	fmt.Println("openning stdout pipe")
	stdoutP, err := os.Open(stdoutPipe)
	if err != nil {
		panic(err)
	}
	fmt.Println("stdout pipe is open")

	fmt.Println("openning stderr pipe")
	stderrP, err := os.Open(stderrPipe)
	if err != nil {
		panic(err)
	}
	fmt.Println("stderr pipe is open")

	// process.Attach
	if processIO.Stdin != nil {
		go io.Copy(stdinP, processIO.Stdin)
	}

	if processIO.Stdout != nil {
		go io.Copy(processIO.Stdout, stdoutP)
	}

	if processIO.Stderr != nil {
		go io.Copy(processIO.Stderr, stderrP)
	}

	return process, nil
}

func (d *DadooExecer) Attach(log lager.Logger, bundlePath, id, processID string, io garden.ProcessIO) (garden.Process, error) {
	return nil, errors.New("not implemented")
}
