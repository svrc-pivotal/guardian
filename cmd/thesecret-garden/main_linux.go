package main

import (
	"fmt"
	"os"
	"os/exec"
)

func main() {
	_, err := exec.Command("mount", "--make-slave", os.Args[1]).CombinedOutput()
	if err != nil {
		fmt.Printf("ERR: %s\n", err.Error())
	}

	cmd := exec.Command(os.Args[2], os.Args[3:]...)
	cmd.Stdout = os.Stdout
	err = cmd.Run()
	if err != nil {
		fmt.Printf("ERR: %s\n", err.Error())
	}
}
