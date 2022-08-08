package main

import (
	"io"

	"github.com/graph-guard/gguard-proxy/cli"
)

func serve(w io.Writer, c cli.CommandServe) {
	fmt.Fprintf(w, "Command 'serve' is not yet supported on Windows\n")
}
