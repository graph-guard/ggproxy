package main

import (
	"io"

	"github.com/graph-guard/gguard-proxy/cli"
)

func stop(w io.Writer, c cli.CommandStop) {
	fmt.Fprintf(w, "command 'stop' is not yet supported on Windows\n")
}
