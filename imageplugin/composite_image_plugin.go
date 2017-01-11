package imageplugin

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"code.cloudfoundry.org/garden"
	"code.cloudfoundry.org/garden-shed/rootfs_provider"
	"code.cloudfoundry.org/lager"
	"github.com/cloudfoundry/gunk/command_runner"
	errorwrapper "github.com/pkg/errors"
)

//go:generate counterfeiter . CommandCreator
type CommandCreator interface {
	CreateCommand(log lager.Logger, handle string, spec rootfs_provider.Spec) (*exec.Cmd, error)
	DestroyCommand(log lager.Logger, handle string) (*exec.Cmd, error)
	MetricsCommand(log lager.Logger, handle string) (*exec.Cmd, error)
	GCCommand(log lager.Logger) (*exec.Cmd, error)
}

type CompositeImagePlugin struct {
	unprivilegedCommandCreator CommandCreator
	commandRunner              command_runner.CommandRunner
}

func New(unprivilegedCommandCreator CommandCreator,
	commandRunner command_runner.CommandRunner) *CompositeImagePlugin {

	return &CompositeImagePlugin{
		unprivilegedCommandCreator: unprivilegedCommandCreator,
		commandRunner:              commandRunner,
	}
}

func (p *CompositeImagePlugin) Create(log lager.Logger, handle string, spec rootfs_provider.Spec) (string, []string, error) {
	createCmd, err := p.unprivilegedCommandCreator.CreateCommand(log, handle, spec)
	if err != nil {
		return "", nil, errorwrapper.Wrap(err, "creating image plugin create-command")
	}

	stdoutBuffer := bytes.NewBuffer([]byte{})
	createCmd.Stdout = stdoutBuffer
	createCmd.Stderr = stdoutBuffer

	log.Info(fmt.Sprintf("WILL %+v", createCmd))
	if err := p.commandRunner.Run(createCmd); err != nil {
		return "", nil, errorwrapper.Wrapf(err, "running image plugin create: %s", stdoutBuffer.String())
	}

	imagePath := strings.TrimSpace(stdoutBuffer.String())
	rootfsPath := filepath.Join(imagePath, "rootfs")

	return rootfsPath, nil, nil
}

func (p *CompositeImagePlugin) Destroy(log lager.Logger, handle string, privileged bool) error {
	return nil
}

func (p *CompositeImagePlugin) Metrics(log lager.Logger, handle string, privileged bool) (garden.ContainerDiskStat, error) {
	return garden.ContainerDiskStat{}, nil
}

func (p *CompositeImagePlugin) GC(log lager.Logger) error {
	return nil
}
