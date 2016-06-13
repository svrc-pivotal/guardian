package driver

import (
	"bytes"
	"fmt"
	"os/exec"

	"github.com/cloudfoundry-incubator/guardian/kawasaki/iptables"
	"github.com/cloudfoundry/gunk/command_runner"
)

type IPTablesController struct {
	iptablesBin string
	runner      command_runner.CommandRunner
}

func New(iptablesBin string, runner command_runner.CommandRunner, chainPrefix string) *IPTablesController {
	return &IPTablesController{
		iptablesBin: iptablesBin,
		runner:      runner,
	}
}

func (iptables *IPTablesController) CreateChain(table, chain string) error {
	return iptables.run("create-instance-chains", exec.Command(iptables.iptablesBin, "--wait", "--table", table, "-N", chain))
}

func (iptables *IPTablesController) DeleteChain(table, chain string) error {
	shellCmd := fmt.Sprintf(
		`iptables --wait --table %s -X %s 2> /dev/null || true`,
		table, chain,
	)
	return iptables.run("delete-instance-chains", exec.Command("sh", "-c", shellCmd))
}

func (iptables *IPTablesController) FlushChain(table, chain string) error {
	shellCmd := fmt.Sprintf(
		`iptables --wait --table %s -F %s 2> /dev/null || true`,
		table, chain,
	)
	return iptables.run("flush-instance-chains", exec.Command("sh", "-c", shellCmd))
}

func (iptables *IPTablesController) DeleteChainReferences(table, targetChain, referencedChain string) error {
	shellCmd := fmt.Sprintf(
		`set -e; /sbin/iptables --wait --table %s -S %s | grep "%s" | sed -e "s/-A/-D/" | xargs --no-run-if-empty --max-lines=1 iptables -w -t %s`,
		table, targetChain, referencedChain, table,
	)
	return iptables.run("delete-referenced-chains", exec.Command("sh", "-c", shellCmd))
}

func (iptables *IPTablesController) ResetChain(table, chain string, rules []iptables.Rule) error {
	iptables.FlushChain(table, chain)

	for _, rule := range rules {
		iptables.AppendRule(table, chain, rule)
	}

	return nil
}

func (iptables *IPTablesController) PrependRule(table, chain string, rule iptables.Rule) error {
	return iptables.run("prepend", exec.Command("/sbin/iptables", append([]string{"-w", "-I", chain, "1", "--table", table}, rule.Flags(chain)...)...))
}

func (iptables *IPTablesController) AppendRule(table, chain string, rule iptables.Rule) error {
	return iptables.run("append", exec.Command("/sbin/iptables", append([]string{"-w", "-A", chain, "--table", table}, rule.Flags(chain)...)...))
}

func (iptables *IPTablesController) ChainExists(table string, chain string) bool {
	cmd := exec.Command("iptables", "-w", "-t", table, "-L", chain)
	return iptables.run("checking-chain-exists", cmd) == nil
}

func (iptables *IPTablesController) run(action string, cmd *exec.Cmd) error {
	var buff bytes.Buffer
	cmd.Stderr = &buff

	if err := iptables.runner.Run(cmd); err != nil {
		return fmt.Errorf("iptables %s: %s", action, buff.String())
	}

	return nil
}
