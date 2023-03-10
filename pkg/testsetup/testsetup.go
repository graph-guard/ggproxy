package testsetup

import (
	"embed"
	"io/fs"
	"path/filepath"

	"github.com/graph-guard/ggproxy/pkg/config"
)

/* SPECIAL NOTE:                                     *\
\* Symlinks are not allowed in embedded filesystems! */

const (
	SetupNameStarwars     = "starwars"
	SetupNameInputsSchema = "inputs_schema"
)

func ByName(name string) (s Setup, ok bool) {
	switch name {
	case SetupNameStarwars:
		return read(fsStarwars, SetupNameStarwars), true
	case SetupNameInputsSchema:
		return read(fsInputsSchema, SetupNameInputsSchema), true
	}
	return s, false
}

//go:embed starwars
var fsStarwars embed.FS

//go:embed inputs_schema
var fsInputsSchema embed.FS

func read(fsys fs.FS, root string) Setup {
	c, err := config.Read(fsys, root, filepath.Join(root, "config.yml"))
	if err != nil {
		panic(err)
	}
	return Setup{
		Name:   root,
		Config: c,
	}
}

type Setup struct {
	Name   string
	Config *config.Config
}
