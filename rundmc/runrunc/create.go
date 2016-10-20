package runrunc

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"code.cloudfoundry.org/garden"
	"code.cloudfoundry.org/lager"
	"github.com/cloudfoundry/gunk/command_runner"
)

type Creator struct {
	runcPath      string
	commandRunner command_runner.CommandRunner
}

func NewCreator(runcPath string, commandRunner command_runner.CommandRunner) *Creator {
	return &Creator{
		runcPath, commandRunner,
	}
}

func (c *Creator) Create(log lager.Logger, bundlePath, id string, _ garden.ProcessIO) (theErr error) {
	log = log.Session("create", lager.Data{"bundle": bundlePath})

	defer log.Info("finished")

	logFilePath := filepath.Join(bundlePath, "create.log")
	pidFilePath := filepath.Join(bundlePath, "pidfile")

	ownAUFSPath, err2 := exec.Command("/bin/bash", "-c", fmt.Sprintf("cat %s | python -m json.tool | grep aufs | cut -d '\"' -f 4", filepath.Join(bundlePath, "config.json"))).CombinedOutput()
	if err2 != nil {
		return err2
	}
	layerID := filepath.Base(string(ownAUFSPath))
	cmd := exec.Command("unshare", "-m", "/bin/bash", "--", "-c", "mount|grep aufs|grep -v \""+layerID[0:len(layerID)-1]+"\"|cut -d ' ' -f 3|xargs -I {} umount {};"+
		c.runcPath+" --debug --log "+logFilePath+" create --no-new-keyring --bundle "+bundlePath+" --pid-file "+pidFilePath+" "+id)
	//cmd := exec.Command(c.runcPath, "--debug", "--log", logFilePath, "create", "--no-new-keyring", "--bundle", bundlePath, "--pid-file", pidFilePath, id)

	log.Info("creating", lager.Data{
		"runc":        c.runcPath,
		"bundlePath":  bundlePath,
		"id":          id,
		"logPath":     logFilePath,
		"pidFilePath": pidFilePath,
	})

	err := c.commandRunner.Run(cmd)

	defer func() {
		theErr = processLogs(log, logFilePath, err)
	}()

	return
}

func processLogs(log lager.Logger, logFilePath string, upstreamErr error) error {
	logReader, err := os.OpenFile(logFilePath, os.O_RDONLY, 0644)
	if err != nil {
		return fmt.Errorf("runc create: open log file '%s': %s", logFilePath, err)
	}

	buff, readErr := ioutil.ReadAll(logReader)
	if readErr != nil {
		return fmt.Errorf("runc create: read log file: %s", readErr)
	}

	forwardRuncLogsToLager(log, buff)

	if upstreamErr != nil {
		return wrapWithErrorFromRuncLog(log, upstreamErr, buff)
	}

	return nil
}
