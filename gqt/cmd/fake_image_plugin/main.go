package main

import (
	"fmt"
	"io/ioutil"
	"os"
)

func main() {
	action := os.Args[2]
	imageID := os.Args[len(os.Args)-1]

	if imageID == "make-it-fail" {
		panic("image-plugin-exploded")
	} else if imageID == "make-it-fail-on-destruction" && action == "delete" {
		panic("image-plugin-exploded-on-destruction")
	}

	err := ioutil.WriteFile(imageID, []byte(fmt.Sprintf("%s", os.Args)), 0777)
	if err != nil {
		panic(err)
	}

	storePath := fmt.Sprintf("/tmp/store-path/%s", imageID)
	if err := os.MkdirAll(storePath, 0777); err != nil {
		panic(err)
	}

	fmt.Printf(storePath)
}
