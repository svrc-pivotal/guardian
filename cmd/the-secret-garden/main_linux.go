package main

import (
	"os"
	"os/exec"
)

func main() {
	_, err := exec.Command("mount", "--make-slave", os.Args[1]).CombinedOutput()
	if err != nil {
		os.Exit(1)
	}

	cmd := exec.Command(os.Args[2], os.Args[3:]...)
	cmd.Stdout = os.Stdout
	if err = cmd.Run(); err != nil {
		os.Exit(1)
	}
}
