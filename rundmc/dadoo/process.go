package dadoo

import (
	"errors"

	"github.com/cloudfoundry-incubator/garden"
)

type Process struct {
	exitWaiter ExitWaiter
	processId  string
	waitSock   string
}

func NewProcess(exitWaiter ExitWaiter, processId, waitSock string) *Process {
	return &Process{
		exitWaiter: exitWaiter,
		processId:  processId,
		waitSock:   waitSock,
	}
}

func (p *Process) ID() string {
	return p.processId
}

func (p *Process) Wait() (int, error) {
	ch, _ := p.exitWaiter.Wait(p.waitSock)
	<-ch
	return -1, nil
}

func (p *Process) SetTTY(garden.TTYSpec) error {
	return errors.New("Not Implemented")
}

func (p *Process) Signal(garden.Signal) error {
	return errors.New("Not Implemented")
}
