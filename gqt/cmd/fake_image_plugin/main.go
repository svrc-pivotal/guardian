package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	action := os.Args[1]
	imageID := os.Args[len(os.Args)-1]

	if imageID == "make-it-fail" {
		panic("image-plugin-exploded")
	} else if imageID == "make-it-fail-on-destruction" && action == "delete" {
		panic("image-plugin-exploded-on-destruction")
	}

	storePath := fmt.Sprintf("/tmp/store-path/%s", imageID)
	if err := os.MkdirAll(storePath, 0777); err != nil {
		panic(err)
	}

	rootFSPath := fmt.Sprintf("%s/rootfs", storePath)
	if err := os.MkdirAll(rootFSPath, 0777); err != nil {
		panic(err)
	}

	whoamiPath := filepath.Join("/tmp/store-path", fmt.Sprintf("%s-whoami-%s", action, imageID))
	uid, err := exec.Command("id", "-u").CombinedOutput()
	if err != nil {
		panic(err)
	}

	gid, err := exec.Command("id", "-g").CombinedOutput()
	if err != nil {
		panic(err)
	}

	err = ioutil.WriteFile(whoamiPath, []byte(fmt.Sprintf("%s - %s\n", strings.Trim(string(uid), "\n"), strings.Trim(string(gid), "\n"))), 0777)
	if err != nil {
		panic(err)
	}

	argsFilepath := filepath.Join("/tmp/store-path", fmt.Sprintf("%s-args-%s", action, imageID))
	err = ioutil.WriteFile(argsFilepath, []byte(fmt.Sprintf("%s", os.Args)), 0777)
	if err != nil {
		panic(err)
	}

	fmt.Printf(storePath)
}
