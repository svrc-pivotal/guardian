package iptables

import (
	"bytes"
	"fmt"
	"os/exec"

	"github.com/cloudfoundry-incubator/guardian/logging"
)

type IPTablesLoggingRunner struct {
	Runner logging.Runner
}

func (r *IPTablesLoggingRunner) Run(action string, cmd *exec.Cmd) error {
	var buff bytes.Buffer
	cmd.Stderr = &buff

	if err := r.Runner.Run(cmd); err != nil {
		return fmt.Errorf("iptables %s: %s", action, buff.String())
	}

	return nil
}
