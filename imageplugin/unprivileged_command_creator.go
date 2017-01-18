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

func (p *UnprivilegedCommandCreator) CreateCommand(log lager.Logger, handle string, spec rootfs_provider.Spec) *exec.Cmd {
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

	return cmd
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
	log = log.Session("image-plugin-metrics", lager.Data{"handle": handle})
	log.Debug("start")
	defer log.Debug("end")

	// imagePath := filepath.Dir(rootfs)
	// args := append(p.extraArgs, "stats", imagePath)
	// cmd := exec.Command(p.binPath, args...)
	// cmd.Stderr = lagregator.NewRelogger(log)
	// outBuffer := bytes.NewBuffer([]byte{})
	// cmd.Stdout = outBuffer

	// if err := p.commandRunner.Run(cmd); err != nil {
	// 	logData := lager.Data{"action": "stats", "stderr": outBuffer.String()}
	// 	log.Error("external-image-manager-result", err, logData)
	// 	return garden.ContainerDiskStat{}, fmt.Errorf("external image manager metrics failed: %s (%s)", outBuffer.String(), err)
	// }

	// var metrics map[string]map[string]uint64
	// if err := json.NewDecoder(outBuffer).Decode(&metrics); err != nil {
	// 	return garden.ContainerDiskStat{}, fmt.Errorf("parsing metrics: %s", err)
	// }

	// return garden.ContainerDiskStat{
	// 	TotalBytesUsed:     metrics["disk_usage"]["total_bytes_used"],
	// 	ExclusiveBytesUsed: metrics["disk_usage"]["exclusive_bytes_used"],
	// }, nil
	//
	return nil
}

func (p *UnprivilegedCommandCreator) GCCommand(log lager.Logger) *exec.Cmd {
	log = log.Session("image-plugin-gc")
	log.Debug("start")
	defer log.Debug("end")

	// args := append(p.extraArgs, "clean")
	// cmd := exec.Command(p.binPath, args...)
	// cmd.Stderr = lagregator.NewRelogger(log)
	// outBuffer := bytes.NewBuffer([]byte{})
	// cmd.Stdout = outBuffer

	// if err := p.commandRunner.Run(cmd); err != nil {
	// 	logData := lager.Data{"action": "clean", "stdout": outBuffer.String()}
	// 	log.Error("external-image-manager-result", err, logData)
	// 	return fmt.Errorf("external image manager clean failed: %s (%s)", outBuffer.String(), err)
	// }

	return nil
}

func stringifyMapping(mapping specs.LinuxIDMapping) string {
	return fmt.Sprintf("%d:%d:%d", mapping.ContainerID, mapping.HostID, mapping.Size)
}
