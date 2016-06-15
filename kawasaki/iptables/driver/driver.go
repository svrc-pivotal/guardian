package driver

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

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
	shellCmd := fmt.Sprintf(
		`%s --wait --table %s -N %s`,
		iptables.iptablesBin, table, chain,
	)
	return iptables.run("create-instance-chains", exec.Command("sh", "-c", shellCmd))
}

func (iptables *IPTablesController) DeleteChain(table, chain string) error {
	shellCmd := fmt.Sprintf(
		`%s --wait --table %s -X %s 2> /dev/null || true`,
		iptables.iptablesBin, table, chain,
	)
	return iptables.run("delete-instance-chains", exec.Command("sh", "-c", shellCmd))
}

func (iptables *IPTablesController) FlushChain(table, chain string) error {
	shellCmd := fmt.Sprintf(
		`%s --wait --table %s -F %s 2> /dev/null || true`,
		iptables.iptablesBin, table, chain,
	)
	return iptables.run("flush-instance-chains", exec.Command("sh", "-c", shellCmd))
}

func (iptables *IPTablesController) ResetChain(table, chain string, rules []iptables.Rule) error {
	shellCmd := fmt.Sprintf(
		`%s --wait --table %s -F %s`,
		iptables.iptablesBin, table, chain,
	)
	if err := iptables.run("flush-instance-chains", exec.Command("sh", "-c", shellCmd)); err != nil {
		return err
	}

	for _, rule := range rules {
		if err := iptables.AppendRule(table, chain, rule); err != nil {
			return err
		}
	}

	return nil
}

func (iptables *IPTablesController) ChainExists(table string, chain string) bool {
	shellCmd := fmt.Sprintf(
		`%s -w -t %s -L %s`,
		iptables.iptablesBin, table, chain,
	)
	cmd := exec.Command("sh", "-c", shellCmd)
	return iptables.run("checking-chain-exists", cmd) == nil
}

func (iptables *IPTablesController) DeleteChainReferences(table, targetChain, referencedChain string) error {
	shellCmd := fmt.Sprintf(
		`set -e; %s --wait --table %s -S %s | grep "%s" | sed -e "s/-A/-D/" | xargs --no-run-if-empty --max-lines=1 iptables -w -t %s`,
		iptables.iptablesBin, table, targetChain, referencedChain, table,
	)
	return iptables.run("delete-referenced-chains", exec.Command("sh", "-c", shellCmd))
}

func (iptables *IPTablesController) AppendRule(table, chain string, rule iptables.Rule) error {
	shellCmd := []string{iptables.iptablesBin, "-w", "-A", chain, "--table", table}
	shellCmd = append(shellCmd, rule.Flags(chain)...)
	return iptables.run("append", exec.Command("sh", "-c", strings.Join(shellCmd, " ")))
}

func (iptables *IPTablesController) PrependRule(table, chain string, rule iptables.Rule) error {
	shellCmd := []string{iptables.iptablesBin, "-w", "-I", chain, "1", "--table", table}
	shellCmd = append(shellCmd, rule.Flags(chain)...)
	return iptables.run("prepend", exec.Command("sh", "-c", strings.Join(shellCmd, " ")))
}

func (iptables *IPTablesController) run(action string, cmd *exec.Cmd) error {
	var buff bytes.Buffer
	cmd.Stderr = &buff

	if err := iptables.runner.Run(cmd); err != nil {
		return fmt.Errorf("iptables %s: %s", action, buff.String())
	}

	return nil
}
