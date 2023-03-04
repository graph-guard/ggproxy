package main

import (
	"io"

	"github.com/graph-guard/ggproxy/pkg/cli"
)

func serve(w io.Writer, c cli.CommandServe) {
	fmt.Fprintf(w, "Command 'serve' is not yet supported on Windows\n")
}
