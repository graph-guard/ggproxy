package main

import (
	"io"

	"github.com/graph-guard/ggproxy/cli"
)

func reload(w io.Writer, c cli.CommandReload) {
	fmt.Fprintf(w, "Command 'reload' is not yet supported on Windows\n")
}
