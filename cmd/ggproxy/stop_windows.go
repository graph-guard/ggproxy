package main

import (
	"io"

	"github.com/graph-guard/ggproxy/cli"
)

func stop(w io.Writer, c cli.CommandStop) {
	fmt.Fprintf(w, "Command 'stop' is not yet supported on Windows\n")
}
