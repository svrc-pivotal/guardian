package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/docker/docker/pkg/reexec"
)

var logLocation = flag.String("log-file", "", "location to write undoo logs")

func init() {
	reexec.Register("namespaced", namespaced)
	if reexec.Init() {
		os.Exit(0)
	}
}

func main() {
	flag.Parse()

	setupLogging()
	reexecInNamespace(flag.Args()...)
}

func reexecInNamespace(args ...string) {
	reexecArgs := append([]string{"namespaced", "-log-file", *logLocation}, args...)
	cmd := reexec.Command(reexecArgs...)

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWNS,
	}

	if err := cmd.Run(); err != nil {
		os.Exit(exitCode(err))
	}
}

func namespaced() {
	flag.Parse()

	if err := unmount(flag.Args()[0], flag.Args()[1]); err != nil {
		writeLog(fmt.Sprintf("undoo failed to perform unmounting: %s, mountsRoot: %s, layerToKeep: %s", err, flag.Args()[1], flag.Args()[2]))
		os.Exit(3)
	}

	cmd := exec.Command(flag.Args()[2], flag.Args()[3:]...)

	if err := cmd.Start(); err != nil {
		writeLog(fmt.Sprintf("undoo failed to execute: %s", err))
		os.Exit(4)
	}

	if err := cmd.Wait(); err != nil {
		os.Exit(exitCode(err))
	}
}

func unmount(mountsRoot, layerToKeep string) error {
	mountsFile, err := os.Open("/proc/mounts")
	if err != nil {
		return err
	}

	mountsReader := bufio.NewReader(mountsFile)
	for {
		lineBytes, _, err := mountsReader.ReadLine()
		if err == io.EOF {
			break
		}
		line := string(lineBytes)
		if strings.Contains(line, mountsRoot) && !strings.Contains(line, layerToKeep) {
			mount := strings.Split(line, " ")[1]

			if mount == mountsRoot {
				continue
			}

			err = syscall.Unmount(mount, syscall.MNT_DETACH)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func exitCode(err error) int {
	if exiterr, ok := err.(*exec.ExitError); ok {
		if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
			return status.ExitStatus()
		}
	}
	return 42
}

func setupLogging() {
	if *logLocation == "" {
		os.Exit(1)
	}
	if err := ioutil.WriteFile(*logLocation, []byte{}, 0666); err != nil {
		os.Exit(2)
	}
}

func writeLog(log string) {
	logFile, err := os.OpenFile(*logLocation, os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		os.Exit(2)
	}
	defer logFile.Close()

	logWriter := bufio.NewWriter(logFile)
	logWriter.WriteString(fmt.Sprintf("msg=%q\n", log))
	logWriter.Flush()
}
