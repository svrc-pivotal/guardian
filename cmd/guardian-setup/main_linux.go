package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/cloudfoundry/gunk/command_runner/linux_command_runner"
	"github.com/jessevdk/go-flags"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/sigmon"

	"code.cloudfoundry.org/guardian/gardener"
	"code.cloudfoundry.org/guardian/guardiancmd"
	"code.cloudfoundry.org/guardian/kawasaki/iptables"
	"code.cloudfoundry.org/guardian/logging"
	locksmithpkg "code.cloudfoundry.org/guardian/pkg/locksmith"
	"code.cloudfoundry.org/guardian/rundmc"
	"code.cloudfoundry.org/lager"
)

type GuardianSetupCommand struct {
	Logger                     guardiancmd.LagerFlag
	ServerTag                  string                 `long:"tag" description:"Optional 2-character identifier used for namespacing global configuration."`
	IPTableBinPath             string                 `long:"iptables-bin"  default:"/sbin/iptables" description:"path to the iptables binary"`
	DestroyContainersOnStartup bool                   `long:"destroy-containers-on-startup" description:"Clean up all the existing containers on startup."`
	AllowHostAccess            bool                   `long:"allow-host-access" description:"Allow network access to the host machine."`
	DenyNetworks               []guardiancmd.CIDRFlag `long:"deny-network"      description:"Network ranges to which traffic from containers will be denied. Can be specified multiple times."`
}

func main() {
	cmd := &GuardianSetupCommand{}

	parser := flags.NewParser(cmd, flags.Default)
	parser.NamespaceDelimiter = "-"

	args, err := parser.Parse()
	if err != nil {
		os.Exit(1)
	}

	err = cmd.Execute(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func (cmd *GuardianSetupCommand) Execute([]string) error {
	return <-ifrit.Invoke(sigmon.New(cmd)).Wait()
}

func (cmd *GuardianSetupCommand) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	logger, _ := cmd.Logger.Logger("guardian-setup")

	starters := []gardener.Starter{
		cmd.wireCgroupStarter(logger),
		cmd.wireIptablesStarter(logger),
	}

	for _, starter := range starters {
		if err := starter.Start(); err != nil {
			return fmt.Errorf("error starting starter: %s", err)
		}
	}
	logger.Info("finished")
	return nil
}

func (cmd *GuardianSetupCommand) wireIptablesStarter(logger lager.Logger) gardener.Starter {
	interfacePrefix := fmt.Sprintf("w%s", cmd.ServerTag)
	chainPrefix := fmt.Sprintf("w-%s-", cmd.ServerTag)
	iptRunner := &logging.Runner{CommandRunner: linux_command_runner.New(), Logger: logger.Session("iptables-runner")}
	locksmith := &locksmithpkg.FileSystem{}
	ipTables := iptables.New(cmd.IPTableBinPath, "iptables-restore-not-needed", iptRunner, locksmith, chainPrefix)

	var denyNetworksList []string
	for _, network := range cmd.DenyNetworks {
		denyNetworksList = append(denyNetworksList, network.String())
	}
	return iptables.NewStarter(ipTables, cmd.AllowHostAccess, interfacePrefix, denyNetworksList, cmd.DestroyContainersOnStartup)
}

func (cmd *GuardianSetupCommand) wireCgroupStarter(logger lager.Logger) gardener.Starter {
	var cgroupsMountpoint string
	if cmd.ServerTag != "" {
		cgroupsMountpoint = filepath.Join(os.TempDir(), fmt.Sprintf("cgroups-%s", cmd.ServerTag))
	} else {
		cgroupsMountpoint = "/sys/fs/cgroup"
	}

	return rundmc.NewStarter(logger, mustOpen("/proc/cgroups"), mustOpen("/proc/self/cgroup"), cgroupsMountpoint, linux_command_runner.New())
}

func mustOpen(path string) io.ReadCloser {
	if r, err := os.Open(path); err != nil {
		panic(err)
	} else {
		return r
	}
}
