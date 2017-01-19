package imageplugin

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"code.cloudfoundry.org/garden"
	"code.cloudfoundry.org/garden-shed/rootfs_provider"
	"code.cloudfoundry.org/lager"
	"github.com/cloudfoundry/gunk/command_runner"
	errorwrapper "github.com/pkg/errors"
	"github.com/tscolari/lagregator"
)

//go:generate counterfeiter . CommandCreator
type CommandCreator interface {
	CreateCommand(log lager.Logger, handle string, spec rootfs_provider.Spec) *exec.Cmd
	DestroyCommand(log lager.Logger, handle string) *exec.Cmd
	MetricsCommand(log lager.Logger, handle string) *exec.Cmd
	GCCommand(log lager.Logger) *exec.Cmd
}

type ImagePlugin struct {
	UnprivilegedCommandCreator CommandCreator
	PrivilegedCommandCreator   CommandCreator
	CommandRunner              command_runner.CommandRunner
	DefaultRootfs              string
}

// func New(UnprivilegedCommandCreator CommandCreator,
// 	PrivilegedCommandCreator CommandCreator,
// 	CommandRunner command_runner.CommandRunner,
// 	DefaultRootfs string) ImagePlugin {

// 	return ImagePlugin{
// 		UnprivilegedCommandCreator: UnprivilegedCommandCreator,
// 		PrivilegedCommandCreator:   PrivilegedCommandCreator,
// 		CommandRunner:              CommandRunner,
// 		DefaultRootfs:              DefaultRootfs,
// 	}
// }

func (p *ImagePlugin) Create(log lager.Logger, handle string, spec rootfs_provider.Spec) (string, []string, error) {
	log = log.Session("image-plugin-create")
	log.Debug("start")
	defer log.Debug("end")

	if spec.RootFS.String() == "" {
		var err error
		spec.RootFS, err = url.Parse(p.DefaultRootfs)

		if err != nil {
			log.Error("parsing-default-rootfs-failed", err)
			return "", nil, errorwrapper.Wrap(err, "parsing default rootfs")
		}
	}

	var createCmd *exec.Cmd
	if spec.Namespaced {
		if p.UnprivilegedCommandCreator == nil {
			return "", nil, errors.New("no image_plugin provided")
		}
		createCmd = p.UnprivilegedCommandCreator.CreateCommand(log, handle, spec)
	} else {
		createCmd = p.PrivilegedCommandCreator.CreateCommand(log, handle, spec)
	}
	stdoutBuffer := bytes.NewBuffer([]byte{})
	createCmd.Stdout = stdoutBuffer
	createCmd.Stderr = lagregator.NewRelogger(log)

	if err := p.CommandRunner.Run(createCmd); err != nil {
		logData := lager.Data{"action": "create", "stdout": stdoutBuffer.String()}
		log.Error("image-plugin-result", err, logData)
		return "", nil, errorwrapper.Wrapf(err, "running image plugin create: %s", stdoutBuffer.String())
	}

	imagePath := strings.TrimSpace(stdoutBuffer.String())
	rootfsPath := filepath.Join(imagePath, "rootfs")

	envVars, err := p.readEnvVars(imagePath)
	if err != nil {
		log.Error("read-image-json-failed", err)
		return "", nil, errorwrapper.Wrap(err, "reading image.json")
	}

	return rootfsPath, envVars, nil
}

func (p *ImagePlugin) Destroy(log lager.Logger, handle string, namespaced bool) error {
	log = log.Session("image-plugin-destroy")
	log.Debug("start")
	defer log.Debug("end")

	var destroyCmd *exec.Cmd
	if namespaced {
		destroyCmd = p.UnprivilegedCommandCreator.DestroyCommand(log, handle)
	} else {
		destroyCmd = p.PrivilegedCommandCreator.DestroyCommand(log, handle)
	}

	stdoutBuffer := bytes.NewBuffer([]byte{})
	destroyCmd.Stdout = stdoutBuffer
	destroyCmd.Stderr = lagregator.NewRelogger(log)

	if err := p.CommandRunner.Run(destroyCmd); err != nil {
		logData := lager.Data{"action": "destroy", "stdout": stdoutBuffer.String()}
		log.Error("image-plugin-result", err, logData)
		return errorwrapper.Wrapf(err, "running image plugin destroy: %s", stdoutBuffer.String())
	}

	return nil
}

func (p *ImagePlugin) Metrics(log lager.Logger, handle string, namespaced bool) (garden.ContainerDiskStat, error) {
	log = log.Session("image-plugin-metrics")
	log.Debug("start")
	defer log.Debug("end")

	var metricsCmd *exec.Cmd
	if namespaced {
		metricsCmd = p.UnprivilegedCommandCreator.MetricsCommand(log, handle)
	} else {
		metricsCmd = p.PrivilegedCommandCreator.MetricsCommand(log, handle)
	}

	stdoutBuffer := bytes.NewBuffer([]byte{})
	metricsCmd.Stdout = stdoutBuffer
	metricsCmd.Stderr = lagregator.NewRelogger(log)

	if err := p.CommandRunner.Run(metricsCmd); err != nil {
		logData := lager.Data{"action": "metrics", "stdout": stdoutBuffer.String()}
		log.Error("image-plugin-result", err, logData)
		return garden.ContainerDiskStat{}, errorwrapper.Wrapf(err, "running image plugin metrics: %s", stdoutBuffer.String())
	}

	// Test that metrics are actually consumed
	return garden.ContainerDiskStat{}, nil
}

func (p *ImagePlugin) GC(log lager.Logger) error {
	return nil
}

func (p *ImagePlugin) readEnvVars(imagePath string) ([]string, error) {
	imageConfigFile, err := os.Open(filepath.Join(imagePath, "image.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}

		return nil, errorwrapper.Wrap(err, "could not open image configuration")
	}

	var imageConfig Image
	if err := json.NewDecoder(imageConfigFile).Decode(&imageConfig); err != nil {
		return nil, errorwrapper.Wrap(err, "parsing image config")
	}

	return imageConfig.Config.Env, nil
}
