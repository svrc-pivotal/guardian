package iptables

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.com/cloudfoundry-incubator/garden"
	"github.com/cloudfoundry/gunk/command_runner"
)

var protocols = map[garden.Protocol]string{
	garden.ProtocolAll:  "all",
	garden.ProtocolTCP:  "tcp",
	garden.ProtocolICMP: "icmp",
	garden.ProtocolUDP:  "udp",
}

//go:generate counterfeiter . Rule
type Rule interface {
	Flags(chain string) []string
}

//go:generate counterfeiter . IPTables
type IPTables interface {
	CreateChain(table, chain string) error
	DeleteChain(table, chain string) error
	FlushChain(table, chain string) error
	DeleteChainReferences(table, targetChain, referencedChain string) error
	PrependRule(chain string, rule Rule) error
	InstanceChain(instanceId string) string
}

type IPTablesController struct {
	runner                                                                                                  command_runner.CommandRunner
	BinPath, preroutingChain, postroutingChain, inputChain, forwardChain, defaultChain, instanceChainPrefix string
}

type Chains struct {
	Prerouting, Postrouting, Input, Forward, Default string
}

func New(BinPath string, runner command_runner.CommandRunner, chainPrefix string) *IPTablesController {
	return &IPTablesController{
		runner: runner,

		BinPath:             BinPath,
		preroutingChain:     chainPrefix + "prerouting",
		postroutingChain:    chainPrefix + "postrouting",
		inputChain:          chainPrefix + "input",
		forwardChain:        chainPrefix + "forward",
		defaultChain:        chainPrefix + "default",
		instanceChainPrefix: chainPrefix + "instance-",
	}
}

func (iptables *IPTablesController) CreateChain(table, chain string) error {
	shellCmd := fmt.Sprintf(
		`%s --wait --table %s -N %s`,
		iptables.BinPath, table, chain,
	)
	return iptables.run("create-instance-chains", exec.Command("sh", "-c", shellCmd))
}

func (iptables *IPTablesController) DeleteChain(table, chain string) error {
	shellCmd := fmt.Sprintf(
		`%s --wait --table %s -X %s 2> /dev/null || true`,
		iptables.BinPath, table, chain,
	)
	return iptables.run("delete-instance-chains", exec.Command("sh", "-c", shellCmd))
}

func (iptables *IPTablesController) FlushChain(table, chain string) error {
	shellCmd := fmt.Sprintf(
		`%s --wait --table %s -F %s 2> /dev/null || true`,
		iptables.BinPath, table, chain,
	)
	return iptables.run("flush-instance-chains", exec.Command("sh", "-c", shellCmd))
}

func (iptables *IPTablesController) DeleteChainReferences(table, targetChain, referencedChain string) error {
	shellCmd := fmt.Sprintf(
		`set -e; %s --wait --table %s -S %s | grep "%s" | sed -e "s/-A/-D/" | xargs --no-run-if-empty --max-lines=1 %s -w -t %s`,
		iptables.BinPath, table, targetChain, referencedChain, iptables.BinPath, table,
	)
	return iptables.run("delete-referenced-chains", exec.Command("sh", "-c", shellCmd))
}

func (iptables *IPTablesController) PrependRule(chain string, rule Rule) error {
	shellCmd := []string{iptables.BinPath, "--wait", "-I", chain, "1"}
	shellCmd = append(shellCmd, rule.Flags(chain)...)
	return iptables.run("prepend", exec.Command("sh", "-c", strings.Join(shellCmd, " ")))
}

func (iptables *IPTablesController) InstanceChain(instanceId string) string {
	return iptables.instanceChainPrefix + instanceId
}

func (iptables *IPTablesController) run(action string, cmd *exec.Cmd) error {
	var buff bytes.Buffer
	cmd.Stderr = &buff

	if err := iptables.runner.Run(cmd); err != nil {
		return fmt.Errorf("%s %s: %s", iptables.BinPath, action, buff.String())
	}

	return nil
}

func (iptables *IPTablesController) appendRule(chain string, rule Rule) error {
	shellCmd := []string{iptables.BinPath, "--wait", "-A", chain}
	shellCmd = append(shellCmd, rule.Flags(chain)...)
	return iptables.run("append", exec.Command("sh", "-c", strings.Join(shellCmd, " ")))
}
