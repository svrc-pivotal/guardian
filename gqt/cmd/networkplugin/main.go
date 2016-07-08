package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

type Properties struct {
	Foo  string `json:"foo"`
	Ping string `json:"ping"`
}

func main() {
	if len(os.Args) < 3 {
		panic("network test plugin requires at least 3 arguments")
	}

	args := strings.Join(os.Args, " ")
	if err := ioutil.WriteFile(os.Args[1], []byte(args), 0700); err != nil {
		panic(err)
	}

	p := &Properties{
		Foo:  "bar",
		Ping: "pong",
	}

	marshaledP, _ := json.Marshal(p)
	fmt.Println(string(marshaledP))
}
