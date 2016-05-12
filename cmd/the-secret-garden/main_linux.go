package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func main() {
	_, err := exec.Command("mount", "--make-slave", os.Args[1]).CombinedOutput()
	if err != nil {
		fmt.Printf("Failed to make slave '%s': %s\n", os.Args[1], err.Error())
		os.Exit(1)
	}

	cmd := exec.Command(os.Args[2], os.Args[3:]...)
	cmd.Stdout = os.Stdout
	if err = cmd.Run(); err != nil {
		fmt.Printf("Failed to execute command '%s': %s\n", strings.Join(os.Args[2:], " "), err.Error())
		os.Exit(1)
	}
}
