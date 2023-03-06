package main

import (
	_ "embed"
	"fmt"
	"log"
	"os"

	"github.com/graph-guard/ggproxy/pkg/config"
	"github.com/graph-guard/ggproxy/pkg/engine/playmon"
	"github.com/graph-guard/ggproxy/pkg/gqlparse"
)

//go:embed request.graphql
var request string

func main() {
	c, err := config.Read(os.DirFS("conf"), "conf", "config.yaml")
	if err != nil {
		log.Fatal("config: ", err)
	}
	e := playmon.New(c.ServicesEnabled[0])
	if e == nil {
		log.Fatal(e)
	}

	fmt.Println("\n\nMATCH:")
	e.Match(
		[]byte(request), nil, nil,
		func(operation, selectionSet []gqlparse.Token) (stop bool) {
			return false
		},
		func(t *config.Template) (stop bool) {
			fmt.Println("MATCHED ", t.ID)
			return false
		},
		func(err error) {
			log.Fatal("ERR:", err)
		},
	)
}
