package playmon_test

import (
	"testing"

	"github.com/graph-guard/ggproxy/config"
	"github.com/graph-guard/ggproxy/engine/playmon"
	"github.com/graph-guard/ggproxy/gqlparse"
	"github.com/graph-guard/ggproxy/testsetup"
	"github.com/graph-guard/gqlscan"
)

var GS string

func BenchmarkMatchStarwars(b *testing.B) {
	s := testsetup.Starwars()
	service := s.Config.ServicesEnabled[0]
	e := playmon.New(service)

	p := gqlparse.NewParser(service.Schema)
	var varvals [][]gqlparse.Token
	var opr gqlscan.Token
	var selset []gqlparse.Token
	src := s.Tests[1].Client.Input.BodyJSON["query"].(string)
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
		e.Match(
			varvals, opr, selset,
			func(t *config.Template) (stop bool) {
				GS = t.ID
				return false
			},
		)
	}
}

func BenchmarkMatchStarwarsWithParser(b *testing.B) {
	s := testsetup.Starwars()
	service := s.Config.ServicesEnabled[0]
	e := playmon.New(service)

	p := gqlparse.NewParser(service.Schema)
	src := []byte(s.Tests[1].Client.Input.BodyJSON["query"].(string))

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		p.Parse(
			src, nil, nil,
			func(
				varValues [][]gqlparse.Token,
				operation, selectionSet []gqlparse.Token,
			) {
				e.Match(
					varValues, operation[0].ID, selectionSet,
					func(t *config.Template) (stop bool) {
						GS = t.ID
						return false
					},
				)
			},
			func(err error) {
				b.Fatalf("parsing request: %v", err)
			},
		)
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
