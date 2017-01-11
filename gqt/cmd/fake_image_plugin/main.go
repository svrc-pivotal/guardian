package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/urfave/cli"
)

func main() {
	fakeImagePlugin := cli.NewApp()
	fakeImagePlugin.Name = "fakeImagePlugin"
	fakeImagePlugin.Usage = "I am FakeImagePlugin!"

	fakeImagePlugin.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "image-path",
			Usage: "Path to use as image",
		},
		cli.StringFlag{
			Name:  "output-path",
			Usage: "Path to write args to",
		},
	}

	fakeImagePlugin.Commands = []cli.Command{
		CreateCommand,
	}

	_ = fakeImagePlugin.Run(os.Args)
}

var CreateCommand = cli.Command{
	Name: "create",
	Action: func(ctx *cli.Context) error {
		outputFile := ctx.GlobalString("output-path")
		err := ioutil.WriteFile(outputFile, []byte(strings.Join(os.Args, " ")), 0777)
		if err != nil {
			panic(err)
		}

		imagePath := ctx.GlobalString("image-path")
		rootFSPath := filepath.Join(imagePath, "rootfs")
		if err := os.MkdirAll(rootFSPath, 0777); err != nil {
			panic(err)
		}

		fmt.Println(imagePath)

		return nil
	},
}
