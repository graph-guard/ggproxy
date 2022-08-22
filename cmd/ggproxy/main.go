package main

import (
	"fmt"
	"os"

	"github.com/graph-guard/ggproxy/cli"
	"github.com/graph-guard/ggproxy/lvs"
)

func main() {
	w := os.Stdout
	switch c := cli.Parse(
		w,
		os.Args,
		func(licenseToken string) bool {
			_, err := lvs.ValidateLicenseToken(licenseToken)
			return err == nil
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
