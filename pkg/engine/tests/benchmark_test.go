package engine_test

// TODO: either reimplement the benchmark or make it playmon compatible.

// import (
// 	"embed"
// 	_ "embed"
// 	"fmt"
// 	"testing"

// 	"github.com/graph-guard/ggproxy/engines/rmap"
// 	"github.com/graph-guard/ggproxy/pkg/gqlparse"
// 	"github.com/graph-guard/gqt"
// )

// var N int

// //go:embed assets/benchassets
// var benchassets embed.FS

// var GS string

// func BenchmarkPartedQuery(b *testing.B) {
// 	templates := readTestAssets(benchassets, "assets/benchassets", "templates")[0].Templates
// 	rules := make(map[string]gqt.Doc, len(templates))
// 	for _, r := range templates {
// 		rules[r.ID] = r.Document
// 	}
// 	rm, _ := rmap.New(rules, 0)

// 	for _, td := range readTestAssets(benchassets, "assets/benchassets", "bench_") {
// 		b.Run(td.ID, func(b *testing.B) {
// 			p := gqlparse.NewParser()
// 			query := []byte(td.Query)
// 			operationName := []byte(td.OperationName)
// 			variables := []byte(td.Variables)
// 			b.ResetTimer()

// 			for n := 0; n < b.N; n++ {
// 				p.Parse(
// 					query, operationName, variables,
// 					func(
// 						varVals [][]gqlparse.Token,
// 						operation []gqlparse.Token,
// 						selectionSet []gqlparse.Token,
// 					) {
// 						rm.MatchAll(
// 							varVals,
// 							operation[0].ID,
// 							selectionSet,
// 							func(id string) { GS = id },
// 						)
// 					}, func(err error) {
// 						panic(fmt.Errorf("unexpected error: %w", err))
// 					},
// 				)
// 			}
// 		})
// 	}
// }
