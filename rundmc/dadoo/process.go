package dadoo

import (
	"errors"
	"os"
	"os/exec"

	"github.com/cloudfoundry-incubator/garden"
)

type Process struct {
	id          string
	command     *exec.Cmd
	exitStatusR *os.File
}

func NewProcess(id string) *Process {
	return &Process{
		id: id,
	}
}

func (p *Process) ID() string {
	return p.id
}

func (p *Process) Wait() (int, error) {
	err := p.command.Wait()

	exitStatusBytes := make([]byte, 1)
	p.exitStatusR.Read(exitStatusBytes)

	return int(exitStatusBytes[0]), err
}

func (p *Process) SetTTY(garden.TTYSpec) error {
	return errors.New("Not Implemented")
}

func (p *Process) Signal(garden.Signal) error {
	return errors.New("Not Implemented")
}

func (p *Process) Start(cmd *exec.Cmd) error {
	exitStatusR, exitStatusW, _ := os.Pipe()

	p.command = cmd
	p.exitStatusR = exitStatusR

	cmd.ExtraFiles = []*os.File{
		exitStatusW,
	}
	return cmd.Start()

}
