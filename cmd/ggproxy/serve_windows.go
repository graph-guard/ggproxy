package main

import (
	"io"

	"github.com/graph-guard/gguard-proxy/cli"
)

func serve(w io.Writer, c cli.CommandServe) {
	_, _ = w.Write([]byte("command 'serve' is not yet supported on Windows"))
}
