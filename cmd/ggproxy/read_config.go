package main

import (
	"io"
	"os"

	"github.com/graph-guard/gguard-proxy/config"
)

func ReadConfig(
	w io.Writer,
	configDirPath string,
) *config.Config {
	conf, err := config.ReadConfig(os.DirFS(configDirPath), ".")
	if err != nil {
		_, _ = w.Write([]byte("reading config: "))
		_, _ = w.Write([]byte(err.Error()))
		_, _ = w.Write([]byte("\n"))
		return nil
	}

	if len(conf.ServicesEnabled) < 1 {
		_, _ = w.Write([]byte("no services enabled\n"))
		return nil
	}

	for i := range conf.ServicesEnabled {
		if len(conf.ServicesEnabled[i].TemplatesEnabled) < 1 {
			_, _ = w.Write([]byte("service "))
			_, _ = w.Write([]byte(conf.ServicesEnabled[i].ID))
			_, _ = w.Write([]byte(" has no templates enabled\n"))
			return nil
		}
	}

	return conf
}
