package iptables

import (
	"bytes"
	"fmt"
	"os/exec"

	"github.com/cloudfoundry/gunk/command_runner"
)

type IPTablesLoggingRunner struct {
	Runner command_runner.CommandRunner
}

func (r *IPTablesLoggingRunner) Run(action string, cmd *exec.Cmd) error {
	var buff bytes.Buffer
	cmd.Stderr = &buff

	if err := r.Runner.Run(cmd); err != nil {
		return fmt.Errorf("iptables %s: %s", action, buff.String())
	}

	return nil
}
