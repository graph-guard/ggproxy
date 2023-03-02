package gqlparse_test

import (
	"testing"

	"github.com/graph-guard/ggproxy/gqlparse"
	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
)

var GI int

func BenchmarkParse(b *testing.B) {
	for _, td := range testdata {
		var schema *ast.Schema
		if td.Data.Schema != "" {
			var err error
			if schema, err = gqlparser.LoadSchema(&ast.Source{
				Name: "schema.graphqls", Input: td.Data.Schema,
			}); err != nil {
				b.Fatalf("parsing schema: %v", err)
			}
		}

		r := gqlparse.NewParser(schema)
		b.Run(td.Decl, func(b *testing.B) {
			src := []byte(td.Data.Src)
			opr := []byte(td.Data.OprName)
			varJSON := []byte(td.Data.VarsJSON)
			b.ResetTimer()
			for n := 0; n < b.N; n++ {
				r.Parse(src, opr, varJSON, func(
					varVals [][]gqlparse.Token,
					opr []gqlparse.Token,
					selectionSet []gqlparse.Token,
				) {
					GI += len(opr) + len(varVals) + len(selectionSet)
				}, func(err error) {
					b.Fatal("unexpected error: ", err)
				})
			}
		})
	}
}

func BenchmarkParseErr(b *testing.B) {
	for _, td := range testsErr {
		var schema *ast.Schema
		if td.Schema != "" {
			var err error
			if schema, err = gqlparser.LoadSchema(&ast.Source{
				Name: "schema.graphqls", Input: td.Schema,
			}); err != nil {
				b.Fatalf("parsing schema: %v", err)
			}
		}

		r := gqlparse.NewParser(schema)
		b.Run(td.Name, func(b *testing.B) {
			src := []byte(td.Src)
			opr := []byte(td.OprName)
			varJSON := []byte(td.VarsJSON)
			b.ResetTimer()
			for n := 0; n < b.N; n++ {
				r.Parse(src, opr, varJSON, func(
					varVals [][]gqlparse.Token,
					opr []gqlparse.Token,
					selectionSet []gqlparse.Token,
				) {
					b.Fatal("unexpected success")
				}, func(err error) {
					GI++
				})
			}
		})
	}
}
