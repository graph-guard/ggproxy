package main

import (
	"fmt"
	"os"

	"github.com/graph-guard/ggproxy/pkg/cli"
)

func main() {
	w := os.Stdout
	switch c := cli.Parse(w, os.Args).(type) {
	case cli.CommandServe:
		serve(w, c)
	case cli.CommandReload:
		reload(w, c)
	case cli.CommandStop:
		stop(w, c)
	default:
		if c != nil {
			panic(fmt.Errorf("unexpected command: %#v", c))
		}
	}
}
