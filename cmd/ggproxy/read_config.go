package main

import (
	"fmt"
	"io"
	"os"

	"github.com/graph-guard/ggproxy/config"
)

func ReadConfig(
	w io.Writer,
	configDirPath string,
) *config.Config {
	conf, err := config.ReadConfig(os.DirFS(configDirPath), ".")
	if err != nil {
		fmt.Fprintf(w, "Reading conf: %s\n", err)
		return nil
	}

	if len(conf.ServicesEnabled) < 1 {
		fmt.Fprintf(w, "No services enabled: %s\n", err)
		return nil
	}

	for i := range conf.ServicesEnabled {
		if len(conf.ServicesEnabled[i].TemplatesEnabled) < 1 {
			fmt.Fprintf(
				w, "service %s has no templates enabled\n",
				conf.ServicesEnabled[i].ID,
			)
			return nil
		}
	}

	return conf
}
