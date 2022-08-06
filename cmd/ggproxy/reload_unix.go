package main

import (
	"io"

	"github.com/graph-guard/gguard-proxy/cli"
)

func reload(w io.Writer, c cli.CommandReload) {
	_, _ = w.Write([]byte("command 'reload' is not yet supported"))
}
