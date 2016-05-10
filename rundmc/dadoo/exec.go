package dadoo

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/cloudfoundry-incubator/garden"
	"github.com/cloudfoundry/gunk/command_runner"
)

type DadooTracker struct {
	runner command_runner.CommandRunner
}

func NewTracker(runner command_runner.CommandRunner) *DadooTracker {
	return &DadooTracker{
		runner: runner,
	}
}

func (d *DadooTracker) Exec(bundlePath, containerId, processId string, processJSON io.Reader, processIO garden.ProcessIO, ttySpec garden.TTYSpec) (garden.Process, error) {
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

	cmd := exec.Command("dadoo", "-stdin", stdinPipe, "-stdout", stdoutPipe, "-stderr", stderrPipe, "exec", "runc", bundlePath, containerId)
	cmd.Stdin = processJSON

	if err := d.runner.Start(cmd); err != nil {
		panic(err)
	}

	stdinP, err := os.OpenFile(stdinPipe, os.O_WRONLY, 0600)
	if err != nil {
		panic(err)
	}

	stdoutP, err := os.Open(stdoutPipe)
	if err != nil {
		panic(err)
	}

	stderrP, err := os.Open(stderrPipe)
	if err != nil {
		panic(err)
	}

	if processIO.Stdin != nil {
		go io.Copy(stdinP, processIO.Stdin)
	}

	if processIO.Stdout != nil {
		go io.Copy(processIO.Stdout, stdoutP)
	}

	if processIO.Stderr != nil {
		go io.Copy(processIO.Stderr, stderrP)
	}

	return nil, nil
}
