package playmon_test

import (
	"testing"

	"github.com/graph-guard/ggproxy/engine/playmon"
	"github.com/graph-guard/ggproxy/gqlparse"
	"github.com/graph-guard/ggproxy/testsetup"
	"github.com/graph-guard/gqlscan"
)

var GS string

func BenchmarkMatchStarwars(b *testing.B) {
	s := testsetup.Starwars()
	e := playmon.New(s.Config.ServicesEnabled[0])

	p := gqlparse.NewParser()
	var varvals [][]gqlparse.Token
	var opr gqlscan.Token
	var selset []gqlparse.Token
	src := s.Tests[0].Client.Input.BodyJSON["query"].(string)
	b.Logf(src)
	p.Parse(
		[]byte(src), nil, nil,
		func(
			varValues [][]gqlparse.Token,
			operation, selectionSet []gqlparse.Token,
		) {
			varvals = make([][]gqlparse.Token, len(varValues))
			for i, vv := range varValues {
				varvals[i] = copyTokens(vv)
			}
			opr = operation[0].ID
			selset = copyTokens(selectionSet)
		},
		func(err error) {
			b.Fatalf("parsing request: %v", err)
		},
	)

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		GS = e.Match(varvals, opr, selset)
	}
}

func copyTokens(original []gqlparse.Token) []gqlparse.Token {
	cp := make([]gqlparse.Token, len(original))
	for i, t := range original {
		v := make([]byte, len(t.Value))
		copy(v, t.Value)
		cp[i] = gqlparse.Token{
			ID:    t.ID,
			Value: v,
		}
	}
	return cp
}
