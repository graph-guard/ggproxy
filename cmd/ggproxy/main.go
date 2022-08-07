package main

import (
	"fmt"
	"os"

	"github.com/graph-guard/gguard-proxy/cli"
)

func main() {
	switch c := cli.Parse(os.Stdout, os.Args).(type) {
	case cli.CommandServe:
		serve(os.Stdout, c)
	case cli.CommandReload:
		reload(os.Stdout, c)
	case cli.CommandStop:
		stop(os.Stdout, c)
	default:
		panic(fmt.Errorf("unexpected command: %#v", c))
	}
}
