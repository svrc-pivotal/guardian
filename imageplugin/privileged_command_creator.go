package imageplugin

import (
	"os/exec"
	"strconv"
	"strings"
	"syscall"

	specs "github.com/opencontainers/runtime-spec/specs-go"

	"code.cloudfoundry.org/garden"
	"code.cloudfoundry.org/garden-shed/rootfs_provider"
	"code.cloudfoundry.org/lager"
)

type PrivilegedCommandCreator struct {
	BinPath    string
	ExtraArgs  []string
	IDMappings []specs.LinuxIDMapping
}

func (p *PrivilegedCommandCreator) CreateCommand(log lager.Logger, handle string, spec rootfs_provider.Spec) *exec.Cmd {
	args := append(p.ExtraArgs, "create")
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
			Uid: 0,
			Gid: 0,
		},
	}

	return cmd

	// 	var args []string
	// 	if spec.Namespaced {
	// 		args = append(p.extraArgs, "create")
	// 	} else {
	// 		args = append(p.privilegedExtraArgs, "create")
	// 	}
	// 	if spec.QuotaSize != 0 {
	// 		if spec.QuotaScope == garden.DiskLimitScopeExclusive {
	// 			args = append(args, "--exclude-image-from-quota")
	// 		}
	// 		args = append(args, "--disk-limit-size-bytes", strconv.FormatInt(spec.QuotaSize, 10))
	// 	}

	// 	if spec.Namespaced {
	// 		for _, mapping := range p.mappings {
	// 			args = append(args, "--uid-mapping", stringifyMapping(mapping))
	// 			args = append(args, "--gid-mapping", stringifyMapping(mapping))
	// 		}
	// 	}

	// 	if spec.RootFS == nil || spec.RootFS.String() == "" {
	// 		args = append(args, p.defaultBaseImage.String())
	// 	} else {
	// 		args = append(args, strings.Replace(spec.RootFS.String(), "#", ":", 1))
	// 	}

	// 	args = append(args, handle)

	// 	cmd := exec.Command(p.binPath, args...)

	// 	cmd.Stderr = lagregator.NewRelogger(log)
	// 	outBuffer := bytes.NewBuffer([]byte{})
	// 	cmd.Stdout = outBuffer

	// 	if spec.Namespaced {
	// 		cmd.SysProcAttr = &syscall.SysProcAttr{
	// 			Credential: &syscall.Credential{
	// 				Uid: p.mappings[0].HostID,
	// 				Gid: p.mappings[0].HostID,
	// 			},
	// 		}
	// 	}

	// if err := p.commandRunner.Run(cmd); err != nil {
	// 	logData := lager.Data{"action": "create", "stdout": outBuffer.String()}
	// 	log.Error("external-image-manager-result", err, logData)
	// 	return "", nil, fmt.Errorf("external image manager create failed: %s (%s)", outBuffer.String(), err)
	// }

	// imagePath := strings.TrimSpace(outBuffer.String())
	// envVars, err := p.readEnvVars(imagePath)
	// if err != nil {
	// 	return "", nil, err
	// }

	// rootFSPath := filepath.Join(imagePath, "rootfs")
	// return rootFSPath, envVars, nil
	// return nil, nil
}

func (p *PrivilegedCommandCreator) DestroyCommand(log lager.Logger, handle string) (*exec.Cmd, error) {
	log = log.Session("image-plugin-destroy", lager.Data{"handle": handle})
	log.Debug("start")
	defer log.Debug("end")

	// 	imagePath := filepath.Dir(rootFSPath)
	// 	args := append(p.extraArgs, "delete", imagePath)
	// 	cmd := exec.Command(p.binPath, args...)

	// 	cmd.Stderr = lagregator.NewRelogger(log)
	// 	outBuffer := bytes.NewBuffer([]byte{})
	// 	cmd.Stdout = outBuffer

	// 	if err := p.commandRunner.Run(cmd); err != nil {
	// 		logData := lager.Data{"action": "delete", "stdout": outBuffer.String()}
	// 		log.Error("external-image-manager-result", err, logData)
	// 		return fmt.Errorf("external image manager destroy failed: %s (%s)", outBuffer.String(), err)
	// 	}

	return nil, nil
}

func (p *PrivilegedCommandCreator) MetricsCommand(log lager.Logger, handle string) (*exec.Cmd, error) {
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
	return nil, nil
}

func (p *PrivilegedCommandCreator) GCCommand(log lager.Logger) (*exec.Cmd, error) {
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

	return nil, nil
}
