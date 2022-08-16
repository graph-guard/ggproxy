package gqlreduce_test

import (
	"testing"

	"github.com/graph-guard/ggproxy/gqlreduce"
)

var GI int

func BenchmarkReduce(b *testing.B) {
	r := gqlreduce.NewReducer()
	for _, td := range testdata {
		b.Run(td.Decl, func(b *testing.B) {
			src := []byte(td.Data.Src)
			opr := []byte(td.Data.OprName)
			varJSON := []byte(td.Data.VarsJSON)
			b.ResetTimer()
			for n := 0; n < b.N; n++ {
				r.Reduce(src, opr, varJSON, func(opr []gqlreduce.Token) {
					GI += len(opr)
				}, func(err error) {
					b.Fatal("unexpected error: ", err)
				})
			}
		})
	}
}

func BenchmarkReduceErr(b *testing.B) {
	r := gqlreduce.NewReducer()
	for _, td := range testdataErr {
		b.Run(td.Decl, func(b *testing.B) {
			src := []byte(td.Data.Src)
			opr := []byte(td.Data.OprName)
			varJSON := []byte(td.Data.VarsJSON)
			b.ResetTimer()
			for n := 0; n < b.N; n++ {
				r.Reduce(src, opr, varJSON, func(opr []gqlreduce.Token) {
					b.Fatal("unexpected success")
				}, func(err error) {
					GI++
				})
			}
		})
	}
}
