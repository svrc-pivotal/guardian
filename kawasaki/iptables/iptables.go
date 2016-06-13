package iptables

import (
	"os/exec"

	"github.com/cloudfoundry-incubator/garden"
)

//go:generate counterfeiter . Rule
type Rule interface {
	Flags(chain string) []string
}

//go:generate counterfeiter . IPTablesDriver
type IPTablesDriver interface {
	ChainExists(table, chain string) bool
	CreateChain(table, chain string) error
	DeleteChain(table, chain string) error
	FlushChain(table, chain string) error
	ResetChain(table, chain string, rules []Rule) error
	DeleteChainReferences(table, targetChain, referencedChain string) error
	PrependRule(table, chain string, rule Rule) error
	AppendRule(table, chain string, rule Rule) error
}

//go:generate counterfeiter . IPTablesRunner
type IPTablesRunner interface {
	Run(action string, cmd *exec.Cmd) error
}

var protocols = map[garden.Protocol]string{
	garden.ProtocolAll:  "all",
	garden.ProtocolTCP:  "tcp",
	garden.ProtocolICMP: "icmp",
	garden.ProtocolUDP:  "udp",
}
