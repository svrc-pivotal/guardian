package iptables

import (
	"fmt"
	"net"
	"os/exec"

	"github.com/pivotal-golang/lager"
)

type InstanceChainCreator struct {
	iptablesBin string
	config      IPTablesConfig
	driver      IPTablesDriver
	runner      IPTablesRunner
}

func NewInstanceChainCreator(iptablesBin string, config IPTablesConfig, driver IPTablesDriver, runner IPTablesRunner) *InstanceChainCreator {
	return &InstanceChainCreator{
		iptablesBin: iptablesBin,
		config:      config,
		driver:      driver,
		runner:      runner,
	}
}

func (cc InstanceChainCreator) Create(logger lager.Logger, instanceId, bridgeName string, ip net.IP, network *net.IPNet) error {
	instanceChain := InstanceChain(cc.config, instanceId)

	if err := cc.driver.CreateChain("nat", instanceChain); err != nil {
		return err
	}

	// Bind nat instance chain to nat prerouting chain
	cmd := exec.Command(cc.iptablesBin, "--wait", "--table", "nat", "-A", cc.config.PreroutingChain, "--jump", instanceChain)
	if err := cc.runner.Run("create-instance-chains", cmd); err != nil {
		return err
	}

	// Enable NAT for traffic coming from containers
	cmd = exec.Command("sh", "-c", fmt.Sprintf(
		`(%s --wait --table nat -S %s | grep "\-j MASQUERADE\b" | grep -q -F -- "-s %s") || %s --wait --table nat -A %s --source %s ! --destination %s --jump MASQUERADE`,
		cc.iptablesBin, cc.config.PostroutingChain, network.String(), cc.iptablesBin,
		cc.config.PostroutingChain, network.String(), network.String(),
	))

	if err := cc.runner.Run("create-instance-chains", cmd); err != nil {
		return err
	}

	// Create filter instance chain
	if err := cc.driver.CreateChain("filter", instanceChain); err != nil {
		return err
	}

	// Allow intra-subnet traffic (Linux ethernet bridging goes through ip stack)
	cmd = exec.Command(cc.iptablesBin, "--wait", "-A", instanceChain, "-s", network.String(), "-d", network.String(), "-j", "ACCEPT")
	if err := cc.runner.Run("create-instance-chains", cmd); err != nil {
		return err
	}

	// Otherwise, use the default filter chain
	cmd = exec.Command(cc.iptablesBin, "--wait", "-A", instanceChain, "--goto", cc.config.DefaultChain)
	if err := cc.runner.Run("create-instance-chains", cmd); err != nil {
		return err
	}

	// Bind filter instance chain to filter forward chain
	cmd = exec.Command(cc.iptablesBin, "--wait", "-I", cc.config.ForwardChain, "2", "--in-interface", bridgeName, "--source", ip.String(), "--goto", instanceChain)
	if err := cc.runner.Run("create-instance-chains", cmd); err != nil {
		return err
	}

	// Create Logging Chain
	return cc.createLoggingChain(logger, instanceId)
}

func (cc InstanceChainCreator) createLoggingChain(logger lager.Logger, instanceId string) error {
	instanceChain := InstanceChain(cc.config, instanceId)
	loggingChain := fmt.Sprintf("%s-log", instanceChain)

	if err := cc.driver.CreateChain("filter", loggingChain); err != nil {
		return err
	}

	cmd := exec.Command(cc.iptablesBin, "--wait", "-A", loggingChain, "-m", "conntrack", "--ctstate", "NEW,UNTRACKED,INVALID", "--protocol", "tcp", "--jump", "LOG", "--log-prefix", instanceId)
	if err := cc.runner.Run("create-instance-chains", cmd); err != nil {
		return err
	}

	cmd = exec.Command(cc.iptablesBin, "--wait", "-A", loggingChain, "--jump", "RETURN")
	if err := cc.runner.Run("create-instance-chains", cmd); err != nil {
		return err
	}

	return nil
}

func (cc InstanceChainCreator) Destroy(logger lager.Logger, instanceId string) error {
	instanceChain := InstanceChain(cc.config, instanceId)

	cc.driver.FlushChain("nat", instanceChain)
	// Prune nat prerouting chain
	cc.driver.DeleteChainReferences("nat", cc.config.PreroutingChain, instanceChain)
	cc.driver.DeleteChain("nat", instanceChain)

	cc.driver.FlushChain("filter", instanceChain)
	// Prune forward chain
	cc.driver.DeleteChainReferences("filter", cc.config.ForwardChain, instanceChain)
	cc.driver.DeleteChain("filter", instanceChain)

	instanceLoggingChain := fmt.Sprintf("%s-log", instanceChain)
	cc.driver.FlushChain("filter", instanceLoggingChain)
	cc.driver.DeleteChain("filter", instanceLoggingChain)

	return nil
}
