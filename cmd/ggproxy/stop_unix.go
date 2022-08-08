package main

import (
	"fmt"
	"io"

	"github.com/graph-guard/gguard-proxy/cli"
)

func stop(w io.Writer, c cli.CommandStop) {
	buf := make([]byte, BufLenCmdSockWrite)
	resp, err := request([]byte("stop"), buf)
	switch err {
	case ErrNoInstanceRunning:
		fmt.Fprintf(w, "No running ggproxy instance detected.\n")
		return
	case nil:
		// OK
	default:
		fmt.Fprintf(w, "error: %s\n", err.Error())
	}
	if string(resp) != "ok" {
		fmt.Fprintf(w, "Unexpected response: %q\n", string(resp))
	}
}
