package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/graph-guard/ggproxy/config"
)

func ReadConfig(
	w io.Writer,
	configPath string,
) *config.Config {
	basePath, fileName := basePathAndFileName(configPath)
	conf, err := config.Read(os.DirFS(basePath), basePath, fileName)
	if err != nil {
		fmt.Fprintf(w, "reading config: %s\n", err)
		return nil
	}

	if len(conf.ServicesEnabled) < 1 {
		fmt.Fprintf(w, "no services enabled: %s\n", err)
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

func basePathAndFileName(path string) (basePath, fileName string) {
	return filepath.Base(path), path[:len(path)-len(filepath.Base(path))]
}
