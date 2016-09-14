package kawasaki

import (
	"net"
	"os"

	"code.cloudfoundry.org/guardian/kawasaki/netns"
	"code.cloudfoundry.org/lager"
)

//go:generate counterfeiter . NetnsExecer
type NetnsExecer interface {
	Exec(netnsFD *os.File, cb func() error) error
}

type configurer struct {
	dnsResolvConfigurer  DnsResolvConfigurer
	hostConfigurer       HostConfigurer
	containerConfigurer  ContainerConfigurer
	instanceChainCreator InstanceChainCreator
	fileOpener           netns.Opener
}

//go:generate counterfeiter . HostConfigurer
type HostConfigurer interface {
	Apply(logger lager.Logger, cfg NetworkConfig, pid int) error
	Destroy(cfg NetworkConfig) error
}

//go:generate counterfeiter . InstanceChainCreator
type InstanceChainCreator interface {
	Create(logger lager.Logger, handle, instanceChain, bridgeName string, ip net.IP, network *net.IPNet) error
	Destroy(logger lager.Logger, instanceChain string) error
}

//go:generate counterfeiter . ContainerConfigurer
type ContainerConfigurer interface {
	Apply(logger lager.Logger, cfg NetworkConfig, pid int) error
}

//go:generate counterfeiter . DnsResolvConfigurer
type DnsResolvConfigurer interface {
	Configure(log lager.Logger, cfg NetworkConfig, pid int) error
}

func NewConfigurer(resolvConfigurer DnsResolvConfigurer, hostConfigurer HostConfigurer, containerConfigurer ContainerConfigurer, instanceChainCreator InstanceChainCreator) *configurer {
	return &configurer{
		dnsResolvConfigurer:  resolvConfigurer,
		hostConfigurer:       hostConfigurer,
		containerConfigurer:  containerConfigurer,
		instanceChainCreator: instanceChainCreator,
	}
}

func (c *configurer) Apply(log lager.Logger, cfg NetworkConfig, pid int) error {
	dnsStop := StartTimer("configurer - c.dnsResolvConfigurer.Configure", log)
	if err := c.dnsResolvConfigurer.Configure(log, cfg, pid); err != nil {
		return err
	}
	dnsStop()

	hostStop := StartTimer("configurer - c.hostConfigurer.Apply", log)
	if err := c.hostConfigurer.Apply(log, cfg, pid); err != nil {
		return err
	}
	hostStop()

	instanceChainStop := StartTimer("configurer - c.instanceChainCreator.Create", log)
	if err := c.instanceChainCreator.Create(log, cfg.ContainerHandle, cfg.IPTableInstance, cfg.BridgeName, cfg.ContainerIP, cfg.Subnet); err != nil {
		return err
	}
	instanceChainStop()

	containerConfigurerStop := StartTimer("configurer - c.containerConfigurer.Apply", log)
	err := c.containerConfigurer.Apply(log, cfg, pid)
	containerConfigurerStop()

	return err
}

func (c *configurer) DestroyBridge(log lager.Logger, cfg NetworkConfig) error {
	return c.hostConfigurer.Destroy(cfg)
}

func (c *configurer) DestroyIPTablesRules(log lager.Logger, cfg NetworkConfig) error {
	return c.instanceChainCreator.Destroy(log, cfg.IPTableInstance)
}
