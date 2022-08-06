package main

import (
	"errors"
	"os"
)

const PIDFilePath = "/var/run/ggproxy.pid"

// getPID reads the /var/run/ggproxy.pid file on UNIX systems
func getPID() string {
	b, err := os.ReadFile(PIDFilePath)
	if errors.Is(err, os.ErrNotExist) {
		return ""
	}
	return string(b)
}
