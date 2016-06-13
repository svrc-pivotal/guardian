package iptables

import "github.com/cloudfoundry-incubator/guardian/kawasaki"

type PortForwarder struct {
	config IPTablesConfig
	driver IPTablesDriver
}

func NewPortForwarder(config IPTablesConfig, driver IPTablesDriver) *PortForwarder {
	return &PortForwarder{
		config: config,
		driver: driver,
	}
}

func (p *PortForwarder) Forward(spec kawasaki.PortForwarderSpec) error {
	return p.driver.AppendRule(
		"nat",
		InstanceChain(p.config, spec.InstanceID),
		ForwardRule{
			DestinationIP:   spec.ExternalIP,
			DestinationPort: spec.FromPort,
			ContainerIP:     spec.ContainerIP,
			ContainerPort:   spec.ToPort,
		},
	)
}
