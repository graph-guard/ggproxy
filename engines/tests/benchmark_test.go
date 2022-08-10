package engines_test

import (
	"context"
	_ "embed"
	"testing"

	"github.com/graph-guard/gguard-proxy/engines/rmap"
	"github.com/graph-guard/gqt"
)

var N int

//go:embed assets/benchassets/rule_00.txt
var benchRule00 string

//go:embed assets/benchassets/rule_01.txt
var benchRule01 string

//go:embed assets/benchassets/rule_02.txt
var benchRule02 string

//go:embed assets/benchassets/rule_03.txt
var benchRule03 string

//go:embed assets/benchassets/rule_04.txt
var benchRule04 string

//go:embed assets/benchassets/rule_05.txt
var benchRule05 string

//go:embed assets/benchassets/rule_06.txt
var benchRule06 string

//go:embed assets/benchassets/rule_07.txt
var benchRule07 string

//go:embed assets/benchassets/rule_08.txt
var benchRule08 string

//go:embed assets/benchassets/query_big.gql
var benchQueryBig string

//go:embed assets/benchassets/query_deep.gql
var benchQueryDeep string

//go:embed assets/benchassets/query_average.gql
var benchQueryAverage string

func BenchmarkRQmap(b *testing.B) {
	var rules []gqt.Doc
	for _, r := range []string{
		benchRule00,
		benchRule01,
		benchRule02,
		benchRule03,
		benchRule04,
		benchRule05,
		benchRule06,
		benchRule07,
		benchRule08,
	} {
		rd, err := gqt.Parse([]byte(r))
		if err.IsErr() {
			panic(err)
		}
		rules = append(rules, rd)
	}
	rm, _ := rmap.New(rules, 0)

	for _, td := range []struct {
		name          string
		query         string
		operationName string
		variablesJSON string
	}{
		{
			name:          "deep",
			query:         benchQueryDeep,
			operationName: "X",
		},
		{
			name:          "big",
			query:         benchQueryBig,
			operationName: "X",
		},
		{
			name:          "average",
			query:         benchQueryAverage,
			operationName: "X",
		},
	} {
		b.Run(td.name, func(b *testing.B) {
			query := []byte(td.query)
			operationName := []byte(td.operationName)
			variablesJSON := []byte(td.variablesJSON)
			b.ResetTimer()

			for n := 0; n < b.N; n++ {
				if err := rm.MatchAll(
					context.Background(),
					query,
					operationName,
					variablesJSON,
					func(n int) { N = n },
				); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}