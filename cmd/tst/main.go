package main

import (
	"fmt"
	"log"

	"github.com/graph-guard/ggproxy/config"
	"github.com/graph-guard/ggproxy/engine/playmon"
	"github.com/graph-guard/ggproxy/gqlparse"
	"github.com/graph-guard/gqt/v4"
	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
)

const (
	schemaSrc     = `type Query {f(a:Int):Int}`
	requestSrc    = `query { f(a:42) }`
	operationName = ``
	variablesJSON = ``
)

var templateSrc = map[string]string{
	"testtemplate": `query { f(a:*) }`,
}

func main() {
	gqlParser := gqlparse.NewParser()
	var schema *ast.Schema
	var gqtParser *gqt.Parser
	if schemaSrc != "" {
		var err error
		if schema, err = gqlparser.LoadSchema(&ast.Source{
			Input: schemaSrc,
			Name:  "schema.graphqls",
		}); err != nil {
			log.Fatalf("parsing schema: %v", err)
		}

		if gqtParser, err = gqt.NewParser([]gqt.Source{
			{
				Name:    "schema.graphqls",
				Content: schemaSrc,
			},
		}); err != nil {
			log.Fatalf("initializing GQT parser: %v", err)
		}
	} else {
		var err error
		if gqtParser, err = gqt.NewParser(nil); err != nil {
			log.Fatalf("initializing GQT parser: %v", err)
		}
	}

	templates := make(map[string]*config.Template, len(templateSrc))
	for id, src := range templateSrc {
		tmpl, _, errs := gqtParser.Parse([]byte(src))
		if errs != nil {
			log.Fatalf("parsing template %q: %v", id, errs)
		}
		templates[id] = &config.Template{
			Enabled:     true,
			ID:          id,
			Name:        "name of " + id,
			Source:      []byte(src),
			GQTTemplate: tmpl,
			FilePath:    "/path/to/" + id + ".gqt",
		}
	}

	s := &config.Service{
		Enabled:    true,
		ID:         "testservice",
		Path:       "/testservice",
		ForwardURL: "https://forward.here",
		Schema:     schema,
		FilePath:   "/path/to/testservice",
		Templates:  templates,
	}
	e := playmon.New(s)
	fmt.Println(e)

	var oprName, varsJSON []byte
	if operationName != "" {
		oprName = []byte(operationName)
	}
	if variablesJSON != "" {
		varsJSON = []byte(variablesJSON)
	}
	gqlParser.Parse(
		[]byte(requestSrc), oprName, varsJSON,
		func(varValues [][]gqlparse.Token, operation, selectionSet []gqlparse.Token) {
			tid := e.Match(varValues, operation[0].ID, selectionSet)
			if tid == "" {
				log.Print("no match")
			}
			fmt.Println("TID:", tid)
		},
		func(err error) {
			log.Fatalf("parsing request: %v", err)
		},
	)
}
