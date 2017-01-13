package main

import (
	"fmt"
	"io"
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
			Name:  "args-path",
			Usage: "Path to write args to",
		},
		cli.StringFlag{
			Name:  "whoami-path",
			Usage: "Path to write uid/gid to",
		},
		cli.StringFlag{
			Name:  "json-file-to-copy",
			Usage: "Path to json file to opy as image.json",
		},
	}

	fakeImagePlugin.Commands = []cli.Command{
		CreateCommand,
	}

	_ = fakeImagePlugin.Run(os.Args)
}

var CreateCommand = cli.Command{
	Name: "create",
	Flags: []cli.Flag{
		cli.StringSliceFlag{
			Name:  "uid-mapping",
			Usage: "uid mappings",
		},
		cli.StringSliceFlag{
			Name:  "gid-mapping",
			Usage: "gid mappings",
		},
		cli.Int64Flag{
			Name:  "disk-limit-size-bytes",
			Usage: "disk limit quota",
		},
		cli.BoolFlag{
			Name:  "exclude-image-from-quota",
			Usage: "exclude base image from disk quota",
		},
	},

	Action: func(ctx *cli.Context) error {
		argsFile := ctx.GlobalString("args-path")
		err := ioutil.WriteFile(argsFile, []byte(strings.Join(os.Args, " ")), 0777)
		if err != nil {
			panic(err)
		}

		whoamiFile := ctx.GlobalString("whoami-path")
		err = ioutil.WriteFile(whoamiFile, []byte(fmt.Sprintf("%d - %d", os.Getuid(), os.Getgid())), 0777)
		if err != nil {
			panic(err)
		}

		imagePath := ctx.GlobalString("image-path")
		rootFSPath := filepath.Join(imagePath, "rootfs")
		if err := os.MkdirAll(rootFSPath, 0777); err != nil {
			panic(err)
		}

		if ctx.GlobalString("json-file-to-copy") != "" {
			if err := copyFile(ctx.GlobalString("json-file-to-copy"), filepath.Join(imagePath, "image.json")); err != nil {
				panic(err)
			}
		}

		fmt.Println(imagePath)

		return nil
	},
}

func copyFile(srcPath, dstPath string) error {
	dirPath := filepath.Dir(dstPath)
	if err := os.MkdirAll(dirPath, 0777); err != nil {
		return err
	}

	reader, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	writer, err := os.Create(dstPath)
	if err != nil {
		reader.Close()
		return err
	}

	if _, err := io.Copy(writer, reader); err != nil {
		writer.Close()
		reader.Close()
		return err
	}

	writer.Close()
	reader.Close()

	return os.Chmod(writer.Name(), 0777)
}
