package gqlparse_test

import (
	"testing"

	"github.com/graph-guard/ggproxy/gqlparse"
)

var GI int

func BenchmarkParse(b *testing.B) {
	r := gqlparse.NewParser()
	for _, td := range testdata {
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
	r := gqlparse.NewParser()
	for _, td := range testdataErr {
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
					b.Fatal("unexpected success")
				}, func(err error) {
					GI++
				})
			}
		})
	}
}
