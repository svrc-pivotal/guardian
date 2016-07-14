package kawasaki

import (
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"

	"github.com/docker/docker/pkg/reexec"

	"code.cloudfoundry.org/guardian/kawasaki/netns"
	"code.cloudfoundry.org/lager"
)

func init() {
	reexec.Register("nsExec", func() {
		var netNsPath, containerIntf, containerIP, bridgeIP, subnet, mtu string
		flag.StringVar(&netNsPath, "netNsPath", "", "netNsPath")
		flag.StringVar(&containerIntf, "containerIntf", "", "containerIntf")
		flag.StringVar(&containerIP, "containerIP", "", "containerIP")
		flag.StringVar(&bridgeIP, "bridgeIP", "", "bridgeIP")
		flag.StringVar(&subnet, "subnet", "", "subnet")
		flag.StringVar(&mtu, "mtu", "", "mtu")
		flag.Parse()

		_, subnetIPNet, _ := net.ParseCIDR(subnet)
		mtuInt, _ := strconv.Atoi(mtu)

		cfg := NetworkConfig{
			ContainerIntf: containerIntf,
			ContainerIP:   net.ParseIP(containerIP),
			BridgeIP:      net.ParseIP(bridgeIP),
			Subnet:        subnetIPNet,
			Mtu:           mtuInt,
		}

		fd, _ := os.Open(netNsPath)
		defer fd.Close()

		netNsExecer := &netns.Execer{}

		c := &configurer{}

		netNsExecer.Exec(fd, func() error {
			return c.containerApplier.Apply(nil, cfg)
		})
	})
}

//go:generate counterfeiter . NetnsExecer
type NetnsExecer interface {
	Exec(netnsFD *os.File, cb func() error) error
}

type configurer struct {
	resolvConfFactory    DnsResolvConfFactory
	hostConfigurer       HostConfigurer
	containerApplier     ContainerApplier
	instanceChainCreator InstanceChainCreator
	fileOpener           netns.Opener
	nsExecer             NetnsExecer
}

//go:generate counterfeiter . HostConfigurer
type HostConfigurer interface {
	Apply(logger lager.Logger, cfg NetworkConfig, netnsFD *os.File) error
	Destroy(cfg NetworkConfig) error
}

//go:generate counterfeiter . InstanceChainCreator
type InstanceChainCreator interface {
	Create(logger lager.Logger, handle, instanceChain, bridgeName string, ip net.IP, network *net.IPNet) error
	Destroy(logger lager.Logger, instanceChain string) error
}

//go:generate counterfeiter . ContainerApplier
type ContainerApplier interface {
	Apply(logger lager.Logger, cfg NetworkConfig) error
}

//go:generate counterfeiter . DnsResolvConfigurer
type DnsResolvConfigurer interface {
	Configure(log lager.Logger) error
}

//go:generate counterfeiter . DnsResolvConfFactory
type DnsResolvConfFactory interface {
	CreateDNSResolvConfigurer(pid int, cfg NetworkConfig) (DnsResolvConfigurer, error)
}

func NewConfigurer(resolvConfFactory DnsResolvConfFactory, hostConfigurer HostConfigurer, containerApplier ContainerApplier, instanceChainCreator InstanceChainCreator, fileOpener netns.Opener, nsExecer NetnsExecer) *configurer {
	return &configurer{
		resolvConfFactory:    resolvConfFactory,
		hostConfigurer:       hostConfigurer,
		containerApplier:     containerApplier,
		instanceChainCreator: instanceChainCreator,
		fileOpener:           fileOpener,
		nsExecer:             nsExecer,
	}
}

func (c *configurer) Apply(log lager.Logger, cfg NetworkConfig, pid int) error {
	dnsResolvConfigurer, err := c.resolvConfFactory.CreateDNSResolvConfigurer(pid, cfg)
	if err != nil {
		return err
	}

	if err := dnsResolvConfigurer.Configure(log); err != nil {
		return err
	}

	fd, err := c.fileOpener.Open(fmt.Sprintf("/proc/%d/ns/net", pid))
	if err != nil {
		return err
	}
	defer fd.Close()

	if err := c.hostConfigurer.Apply(log, cfg, fd); err != nil {
		return err
	}

	if err := c.instanceChainCreator.Create(log, cfg.ContainerHandle, cfg.IPTableInstance, cfg.BridgeName, cfg.ContainerIP, cfg.Subnet); err != nil {
		return err
	}

	cmd := reexec.Command("nsExec",
		"-netNsPath", fmt.Sprintf("/proc/%d/ns/net", pid),
		"-containerIntf", cfg.ContainerIntf,
		"-containerIP", cfg.ContainerIP.String(),
		"-bridgeIP", cfg.BridgeIP.String(),
		"-subnet", cfg.Subnet.String(),
		"-mtu", string(cfg.Mtu),
	)

	if err := cmd.Start(); err != nil {
		return err
	}

	return cmd.Wait()
}

func (c *configurer) DestroyBridge(log lager.Logger, cfg NetworkConfig) error {
	return c.hostConfigurer.Destroy(cfg)
}

func (c *configurer) DestroyIPTablesRules(log lager.Logger, cfg NetworkConfig) error {
	return c.instanceChainCreator.Destroy(log, cfg.IPTableInstance)
}
