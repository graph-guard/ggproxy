package main

import (
	"fmt"
	"io"

	"github.com/graph-guard/ggproxy/pkg/cli"
)

func reload(w io.Writer, c cli.CommandReload) {
	fmt.Fprintf(w, "Command 'reload' is not yet supported\n")
}
