package main

import (
	"fmt"
	"os"

	"github.com/graph-guard/ggproxy/cli"
)

func main() {
	w := os.Stdout
	switch c := cli.Parse(
		w,
		os.Args,
		func(licenceKey string) bool {
			return licenceKey == "CLOSEDBETAAUG2022"
		},
	).(type) {
	case cli.CommandServe:
		serve(w, c)
	case cli.CommandReload:
		reload(w, c)
	case cli.CommandStop:
		stop(w, c)
	default:
		panic(fmt.Errorf("unexpected command: %#v", c))
	}
}
