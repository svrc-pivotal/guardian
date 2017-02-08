package guardiancmd

import (
	"fmt"
	"os"
	"path/filepath"

	"code.cloudfoundry.org/guardian/gardener"
	"code.cloudfoundry.org/guardian/kawasaki/iptables"
	"code.cloudfoundry.org/guardian/logging"
	locksmithpkg "code.cloudfoundry.org/guardian/pkg/locksmith"
	"code.cloudfoundry.org/guardian/rundmc"
	"code.cloudfoundry.org/lager"
	"github.com/cloudfoundry/gunk/command_runner/linux_command_runner"
)

type SetupCommand struct {
	LogLevel                   LagerFlag
	Logger                     lager.Logger
	Tag                        string     `long:"tag" description:"Optional 2-character identifier used for namespacing global configuration."`
	DestroyContainersOnStartup bool       `long:"destroy-containers-on-startup" description:"Clean up all the existing containers on startup."`
	IPTables                   FileFlag   `long:"iptables-bin"  default:"/sbin/iptables" description:"path to the iptables binary"`
	DenyNetworks               []CIDRFlag `long:"deny-network"      description:"Network ranges to which traffic from containers will be denied. Can be specified multiple times."`
	AllowHostAccess            bool       `long:"allow-host-access" description:"Allow network access to the host machine."`
}

func (cmd *SetupCommand) Execute(args []string) error {
	cmd.Logger, _ = cmd.LogLevel.Logger("guardian-setup")

	bulkStarter := BulkStarter{
		Starters: []gardener.Starter{
			cmd.wireCgroupStarter(),
			cmd.wireIPTablesStarter(),
		},
	}

	return bulkStarter.StartAll()
}

type BulkStarter struct {
	Starters []gardener.Starter
}

func (b *BulkStarter) StartAll() error {
	for _, s := range b.Starters {
		if err := s.Start(); err != nil {
			return err
		}
	}
	return nil
}

func (cmd *SetupCommand) wireCgroupStarter() gardener.Starter {
	var cgroupsMountpoint string
	if cmd.Tag != "" {
		cgroupsMountpoint = filepath.Join(os.TempDir(), fmt.Sprintf("cgroups-%s", cmd.Tag))
	} else {
		cgroupsMountpoint = "/sys/fs/cgroup"
	}

	return rundmc.NewStarter(cmd.Logger, mustOpen("/proc/cgroups"), mustOpen("/proc/self/cgroup"), cgroupsMountpoint, linux_command_runner.New())
}

func (cmd *SetupCommand) wireIPTablesStarter() gardener.Starter {
	var denyNetworksList []string
	for _, network := range cmd.DenyNetworks {
		denyNetworksList = append(denyNetworksList, network.String())
	}

	interfacePrefix := fmt.Sprintf("w%s", cmd.Tag)
	chainPrefix := fmt.Sprintf("w-%s-", cmd.Tag)
	iptRunner := &logging.Runner{CommandRunner: linux_command_runner.New(), Logger: cmd.Logger.Session("iptables-runner")}
	locksmith := &locksmithpkg.FileSystem{}
	ipTables := iptables.New(cmd.IPTables.Path(), "iptables-restore-not-used", iptRunner, locksmith, chainPrefix)
	ipTablesStarter := iptables.NewStarter(ipTables, cmd.AllowHostAccess, interfacePrefix, denyNetworksList, cmd.DestroyContainersOnStartup)

	return ipTablesStarter
}
