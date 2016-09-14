package iptables

import (
	"fmt"
	"net"
	"os/exec"

	"code.cloudfoundry.org/guardian/kawasaki"
	"code.cloudfoundry.org/lager"
)

type InstanceChainCreator struct {
	iptables *IPTablesController
}

func NewInstanceChainCreator(iptables *IPTablesController) *InstanceChainCreator {
	return &InstanceChainCreator{
		iptables: iptables,
	}
}

func (cc *InstanceChainCreator) Create(logger lager.Logger, handle, instanceId, bridgeName string, ip net.IP, network *net.IPNet) error {
	instanceChain := cc.iptables.InstanceChain(instanceId)

	createNatChainStop := kawasaki.StartTimer("instanceChainCreator - cc.iptables.CreateChain nat", logger)
	if err := cc.iptables.CreateChain("nat", instanceChain); err != nil {
		return err
	}
	createNatChainStop()

	// Bind nat instance chain to nat prerouting chain
	cmd := exec.Command(cc.iptables.binPath, "--wait", "--table", "nat", "-A", cc.iptables.preroutingChain, "--jump", instanceChain)
	bindInstanceChainStop := kawasaki.StartTimer(fmt.Sprintf("instanceChainCreator - cc.iptables.run %#v", cmd.Args), logger)
	if err := cc.iptables.run("create-instance-chains", cmd); err != nil {
		return err
	}
	bindInstanceChainStop()

	// Enable NAT for traffic coming from containers
	cmd = exec.Command("sh", "-c", fmt.Sprintf(
		`(%s --wait --table nat -S %s | grep "\-j MASQUERADE\b" | grep -q -F -- "-s %s") || %s --wait --table nat -A %s --source %s ! --destination %s --jump MASQUERADE`,
		cc.iptables.binPath, cc.iptables.postroutingChain, network.String(), cc.iptables.binPath, cc.iptables.postroutingChain,
		network.String(), network.String(),
	))
	natStop := kawasaki.StartTimer(fmt.Sprintf("instanceChainCreator - cc.iptables.run %#v", cmd.Args), logger)
	if err := cc.iptables.run("create-instance-chains", cmd); err != nil {
		return err
	}
	natStop()

	// Create filter instance chain
	createFilterChainStop := kawasaki.StartTimer("instanceChainCreator - cc.iptables.CreateChain filter", logger)
	if err := cc.iptables.CreateChain("filter", instanceChain); err != nil {
		return err
	}
	createFilterChainStop()

	// Allow intra-subnet traffic (Linux ethernet bridging goes through ip stack)
	cmd = exec.Command(cc.iptables.binPath, "--wait", "-A", instanceChain, "-s", network.String(), "-d", network.String(), "-j", "ACCEPT")
	intraSubnetStop := kawasaki.StartTimer(fmt.Sprintf("instanceChainCreator - cc.iptables.run %#v", cmd.Args), logger)
	if err := cc.iptables.run("create-instance-chains", cmd); err != nil {
		return err
	}
	intraSubnetStop()

	// Otherwise, use the default filter chain
	cmd = exec.Command(cc.iptables.binPath, "--wait", "-A", instanceChain, "--goto", cc.iptables.defaultChain)
	defaultFilterStop := kawasaki.StartTimer(fmt.Sprintf("instanceChainCreator - cc.iptables.run %#v", cmd.Args), logger)
	if err := cc.iptables.run("create-instance-chains", cmd); err != nil {
		return err
	}
	defaultFilterStop()

	// Bind filter instance chain to filter forward chain
	cmd = exec.Command(cc.iptables.binPath, "--wait", "-I", cc.iptables.forwardChain, "2", "--in-interface", bridgeName, "--source", ip.String(), "--goto", instanceChain)
	if err := cc.iptables.run("create-instance-chains", cmd); err != nil {
		return err
	}

	// Create Logging Chain
	createLoggingChainStop := kawasaki.StartTimer("instanceChainCreator - cc.iptables.CreateChain logging", logger)
	err := cc.createLoggingChain(logger, handle, instanceId)
	createLoggingChainStop()

	return err
}

func (cc *InstanceChainCreator) createLoggingChain(logger lager.Logger, handle, instanceId string) error {
	instanceChain := cc.iptables.InstanceChain(instanceId)
	loggingChain := fmt.Sprintf("%s-log", instanceChain)

	if err := cc.iptables.CreateChain("filter", loggingChain); err != nil {
		return err
	}

	if len(handle) > 29 {
		handle = handle[0:29]
	}

	cmd := exec.Command(cc.iptables.binPath, "--wait", "-A", loggingChain, "-m", "conntrack", "--ctstate", "NEW,UNTRACKED,INVALID", "--protocol", "tcp", "--jump", "LOG", "--log-prefix", handle)
	if err := cc.iptables.run("create-instance-chains", cmd); err != nil {
		return err
	}

	cmd = exec.Command(cc.iptables.binPath, "--wait", "-A", loggingChain, "--jump", "RETURN")
	if err := cc.iptables.run("create-instance-chains", cmd); err != nil {
		return err
	}

	return nil
}

func (cc *InstanceChainCreator) Destroy(logger lager.Logger, instanceId string) error {
	instanceChain := cc.iptables.InstanceChain(instanceId)

	// Prune nat prerouting chain
	cmd := exec.Command("sh", "-c", fmt.Sprintf(
		`%s --wait --table nat -S %s 2> /dev/null | grep "\-j %s\b" | sed -e "s/-A/-D/" | xargs --no-run-if-empty --max-lines=1 %s --wait --table nat`,
		cc.iptables.binPath, cc.iptables.preroutingChain, instanceChain, cc.iptables.binPath,
	))
	if err := cc.iptables.run("prune-prerouting-chain", cmd); err != nil {
		return err
	}

	// Flush instance chain
	if err := cc.iptables.FlushChain("nat", instanceChain); err != nil {
		return err
	}

	// Delete nat instance chain
	if err := cc.iptables.DeleteChain("nat", instanceChain); err != nil {
		return err
	}

	// Prune forward chain
	cmd = exec.Command("sh", "-c", fmt.Sprintf(
		`%s --wait -S %s 2> /dev/null | grep "\-g %s\b" | sed -e "s/-A/-D/" | xargs --no-run-if-empty --max-lines=1 %s --wait`,
		cc.iptables.binPath, cc.iptables.forwardChain, instanceChain, cc.iptables.binPath,
	))
	if err := cc.iptables.run("prune-forward-chain", cmd); err != nil {
		return err
	}

	// Flush instance chain
	cc.iptables.FlushChain("filter", instanceChain)

	// delete instance chain
	cc.iptables.DeleteChain("filter", instanceChain)

	// delete the logging chain
	instanceLoggingChain := fmt.Sprintf("%s-log", instanceChain)
	cc.iptables.FlushChain("filter", instanceLoggingChain)
	cc.iptables.DeleteChain("filter", instanceLoggingChain)

	return nil
}
