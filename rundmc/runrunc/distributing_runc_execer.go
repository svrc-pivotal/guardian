package runrunc

import (
	"github.com/cloudfoundry-incubator/garden"
	"github.com/pivotal-golang/lager"
)

type DistributingRuncExecer struct {
	dadooExecer    RuncExecer
	iodaemonExecer RuncExecer
}

func NewDistributingExecer(iodaemonExecer, dadooExecer RuncExecer) *DistributingRuncExecer {
	return &DistributingRuncExecer{
		dadooExecer:    dadooExecer,
		iodaemonExecer: iodaemonExecer,
	}
}

func (d *DistributingRuncExecer) Exec(log lager.Logger, bundlePath, handle string, spec garden.ProcessSpec, io garden.ProcessIO) (garden.Process, error) {
	if spec.TTY != nil {
		return d.iodaemonExecer.Exec(log, bundlePath, handle, spec, io)
	}
	return d.dadooExecer.Exec(log, bundlePath, handle, spec, io)
}

func (d *DistributingRuncExecer) Attach(log lager.Logger, bundlePath, handle, processID string, io garden.ProcessIO) (garden.Process, error) {
	return d.iodaemonExecer.Attach(log, bundlePath, handle, processID, io)
}
