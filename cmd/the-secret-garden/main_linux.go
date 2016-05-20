package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"github.com/docker/docker/pkg/reexec"
)

func init() {
	reexec.Register("namespaced", namespaced)

	if reexec.Init() {
		os.Exit(0)
	}
}

func namespaced() {
	dataDir := os.Args[1]
	mustRun(exec.Command("mount", "--make-slave", dataDir))

	programPath, err := exec.LookPath(os.Args[5])
	if err != nil {
		fmt.Printf("failed to look path in namespace : %s\n", err)
		os.Exit(1)
	}

	if err := syscall.Exec(programPath, os.Args[5:], os.Environ()); err != nil {
		fmt.Printf("exec failed in namespace: %s\n", err)
		os.Exit(1)
	}
}

func main() {
	dataDir := os.Args[1]
	realGraphParentDir := os.Args[2]
	realGraphDir := os.Args[3]
	graphDir := os.Args[4]

	mustBindMountOnce(dataDir, dataDir)

	mustRun(exec.Command("mount", "--make-shared", dataDir))

	mustRun(exec.Command("chmod", "go-x", realGraphParentDir))

	mustBindMountOnce(realGraphDir, graphDir)

	reexecInNamespace(os.Args[1:]...)
}

func reexecInNamespace(args ...string) {
	reexecArgs := append([]string{"namespaced"}, args...)
	cmd := reexec.Command(reexecArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWNS,
		Pdeathsig:  syscall.SIGKILL,
	}
	forwardSignals(cmd, syscall.SIGTERM)

	if err := cmd.Run(); err != nil {
		fmt.Printf("secret garden exec failed: %s\n", err)
		os.Exit(1)
	}
}

func mustBindMountOnce(srcDir, dstDir string) {
	mounts := mustRun(exec.Command("mount"))
	alreadyMounted := strings.Contains(mounts, fmt.Sprintf("%s on %s", srcDir, dstDir))

	if !alreadyMounted {
		mustRun(exec.Command("mount", "--bind", srcDir, dstDir))
	}
}

func run(cmd *exec.Cmd) error {
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%s: %s: %s", cmd.Path, err, string(out))
	}

	return nil
}

func mustRun(cmd *exec.Cmd) string {
	out, err := cmd.CombinedOutput()
	if err != nil {
		panic(fmt.Sprintf("%s: %s: %s", cmd.Path, err, string(out)))
	}

	return string(out)
}

func forwardSignals(cmd *exec.Cmd, signals ...os.Signal) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, signals...)
	go func() {
		cmd.Process.Signal(<-c)
	}()
}
