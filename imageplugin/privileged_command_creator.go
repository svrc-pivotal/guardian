package imageplugin

import (
	"os/exec"
	"strconv"
	"strings"
	"syscall"

	"code.cloudfoundry.org/garden"
	"code.cloudfoundry.org/garden-shed/rootfs_provider"
	"code.cloudfoundry.org/lager"
)

type PrivilegedCommandCreator struct {
	BinPath   string
	ExtraArgs []string
}

func (cc *PrivilegedCommandCreator) CreateCommand(log lager.Logger, handle string, spec rootfs_provider.Spec) (*exec.Cmd, error) {
	args := append(cc.ExtraArgs, "create")
	if spec.QuotaSize > 0 {
		args = append(args, "--disk-limit-size-bytes", strconv.FormatInt(spec.QuotaSize, 10))

		if spec.QuotaScope == garden.DiskLimitScopeExclusive {
			args = append(args, "--exclude-image-from-quota")
		}
	}

	rootfs := strings.Replace(spec.RootFS.String(), "#", ":", 1)

	args = append(args, rootfs, handle)
	cmd := exec.Command(cc.BinPath, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Credential: &syscall.Credential{
			Uid: 0,
			Gid: 0,
		},
	}

	return cmd, nil
}

func (cc *PrivilegedCommandCreator) DestroyCommand(log lager.Logger, handle string) *exec.Cmd {
	cmd := exec.Command(cc.BinPath, append(cc.ExtraArgs, "delete", handle)...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Credential: &syscall.Credential{
			Uid: 0,
			Gid: 0,
		},
	}

	return cmd
}

func (cc *PrivilegedCommandCreator) MetricsCommand(log lager.Logger, handle string) *exec.Cmd {
	cmd := exec.Command(cc.BinPath, append(cc.ExtraArgs, "stats", handle)...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Credential: &syscall.Credential{
			Uid: 0,
			Gid: 0,
		},
	}

	return cmd
}
