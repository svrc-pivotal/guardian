package imageplugin

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"syscall"

	specs "github.com/opencontainers/runtime-spec/specs-go"

	"code.cloudfoundry.org/garden"
	"code.cloudfoundry.org/garden-shed/rootfs_provider"
	"code.cloudfoundry.org/lager"
)

type UnprivilegedCommandCreator struct {
	BinPath    string
	ExtraArgs  []string
	IDMappings []specs.LinuxIDMapping
}

func (p *UnprivilegedCommandCreator) CreateCommand(log lager.Logger, handle string, spec rootfs_provider.Spec) (*exec.Cmd, error) {
	args := append(p.ExtraArgs, "create")

	for _, mapping := range p.IDMappings {
		args = append(args, "--uid-mapping", stringifyMapping(mapping))
		args = append(args, "--gid-mapping", stringifyMapping(mapping))
	}

	if spec.QuotaSize > 0 {
		args = append(args, "--disk-limit-size-bytes", strconv.FormatInt(spec.QuotaSize, 10))

		if spec.QuotaScope == garden.DiskLimitScopeExclusive {
			args = append(args, "--exclude-image-from-quota")
		}
	}

	rootfs := strings.Replace(spec.RootFS.String(), "#", ":", 1)

	args = append(args, rootfs, handle)
	cmd := exec.Command(p.BinPath, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Credential: &syscall.Credential{
			Uid: p.IDMappings[0].HostID,
			Gid: p.IDMappings[0].HostID,
		},
	}

	return cmd, nil
}

func (p *UnprivilegedCommandCreator) DestroyCommand(log lager.Logger, handle string) *exec.Cmd {
	cmd := exec.Command(p.BinPath, append(p.ExtraArgs, "delete", handle)...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Credential: &syscall.Credential{
			Uid: p.IDMappings[0].HostID,
			Gid: p.IDMappings[0].HostID,
		},
	}

	return cmd
}

func (p *UnprivilegedCommandCreator) MetricsCommand(log lager.Logger, handle string) *exec.Cmd {
	cmd := exec.Command(p.BinPath, append(p.ExtraArgs, "stats", handle)...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Credential: &syscall.Credential{
			Uid: p.IDMappings[0].HostID,
			Gid: p.IDMappings[0].HostID,
		},
	}
	return cmd
}

func (p *UnprivilegedCommandCreator) GCCommand(log lager.Logger) *exec.Cmd {
	return nil
}

func stringifyMapping(mapping specs.LinuxIDMapping) string {
	return fmt.Sprintf("%d:%d:%d", mapping.ContainerID, mapping.HostID, mapping.Size)
}
