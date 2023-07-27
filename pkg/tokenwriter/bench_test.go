package tokenwriter_test

import (
	"io"
	"testing"

	"github.com/graph-guard/ggproxy/pkg/gqlparse"
	"github.com/graph-guard/ggproxy/pkg/tokenwriter"
)

func BenchmarkWrite(b *testing.B) {
	for _, td := range testdata {
		b.Run("", func(b *testing.B) {
			var opr []gqlparse.Token
			r := gqlparse.NewParser(nil)
			r.Parse(
				[]byte(td.Request),
				[]byte(td.OperationName),
				[]byte(td.VariablesJSON),
				func(
					variableValues [][]gqlparse.Token,
					operation []gqlparse.Token,
					selectionSet []gqlparse.Token,
				) {
					opr = make([]gqlparse.Token, len(operation))
					copy(opr, operation)
				},
				func(err error) {
					b.Fatalf("unexpected parser error: %v", err)
				},
			)
			b.ResetTimer()
			for n := 0; n < b.N; n++ {
				if err := tokenwriter.Write(io.Discard, opr); err != nil {
					b.Fatalf("unexpected write error: %v", err)
				}
			}
		})
	}
}
