package playmon_test

// import (
// 	"testing"

// 	"github.com/graph-guard/ggproxy/pkg/config"
// 	"github.com/graph-guard/ggproxy/pkg/engine/playmon"
// 	"github.com/graph-guard/ggproxy/pkg/gqlparse"
// 	"github.com/graph-guard/ggproxy/pkg/testsetup"
// )

// // var GS string

// // func BenchmarkMatchStarwars(b *testing.B) {
// // 	s := testsetup.Starwars()
// // 	service := s.Config.ServicesEnabled[0]
// // 	e := playmon.New(service)
// // 	b.ResetTimer()
// // 	for n := 0; n < b.N; n++ {
// // 		e.Match(
// // 			[]byte(s.Tests[1].Client.Input.BodyJSON["query"].(string)),
// // 			nil, nil,
// // 			func(operation, selectionSet []gqlparse.Token) (stop bool) {
// // 				return false
// // 			},
// // 			func(t *config.Template) (stop bool) {
// // 				GS = t.ID
// // 				return false
// // 			},
// // 			func(err error) {
// // 				b.Fatal("unexpected error:", err)
// // 			},
// // 		)
// // 	}
// // }
