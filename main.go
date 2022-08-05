package main

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"time"

	"github.com/graph-guard/gguard-proxy/server"
	"github.com/graph-guard/gguard/engines/rmap"
	"github.com/graph-guard/gqt"
	plog "github.com/phuslu/log"
	"gopkg.in/yaml.v3"
)

//go:embed server/assets/benchassets
var assets embed.FS

func main() {
	fs.WalkDir(assets, "server/assets/benchassets/queries", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			query, err := fs.ReadFile(assets, path)
			if err != nil {
				panic(err)
			}
			data := make(map[string]string)
			err = yaml.Unmarshal(query, &data)
			if err != nil {
				panic(err)
			}
		}

		return nil
	})

	rawRules := [][]byte{
		[]byte(`
			mutation {
				a {
					a0(
						a0_0: val = [ val <= 0 ] && val != [ val = -1 ]
					)
				}
			}
			`,
		),
	}
	rules := make([]gqt.Doc, 1)
	for i, r := range rawRules {
		rd, err := gqt.Parse(r)
		if err.IsErr() {
			fmt.Println(err)
		}
		rules[i] = rd
	}
	rm, _ := rmap.New(rules, 0)

	log := plog.Logger{
		Level:      plog.DebugLevel,
		TimeField:  "time",
		TimeFormat: "15:04:05",
		Writer:     &plog.IOWriter{os.Stdout},
	}
	// log := zerolog.New(os.Stdout).With().Timestamp().Logger().With().Logger()
	// log := zerolog.New(os.Stdout).With().Timestamp().Logger().With().Caller().Logger()
	// log = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
	services := []server.ServiceConfig{
		{
			Name:        "service_a",
			Source:      "/service_a",
			Destination: "localhost:8080",
			Engine:      rm,
		},
	}

	s := server.New(
		services,
		"proxy",
		":8000",
		true,
		time.Second*10,
		log,
		// log.Level(zerolog.DebugLevel),
	)

	s.Serve()

	// pt := trie.New()
	// pt.Add("/api/endpoint_a", 0)
	// pt.Add("/api/endpoint_b", 1)
	// fmt.Println(pt.HasKeysWithPrefix("/api/endpoint_a/woho?a=1&b=2"))

	// r := radix.New()
	// r.Insert("/api/endpoint_a", 0)
	// r.Insert("/api/endpoint_b", 1)
	// fmt.Println(r.LongestPrefix("/api/endpoint_a/woho?a=1&b=2"))
}
