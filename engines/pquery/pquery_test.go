package pquery_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/graph-guard/ggproxy/engines/pquery"
	"github.com/graph-guard/ggproxy/gqlparse"
	"github.com/graph-guard/ggproxy/utilities/container/hamap"
	"github.com/graph-guard/ggproxy/utilities/xxhash"
	"github.com/stretchr/testify/require"
)

func TestNewQueryPart(t *testing.T) {
	for _, td := range []struct {
		query         string
		operationName string
		variablesJSON string
		expect        []pquery.QueryPart
	}{
		{
			operationName: "X",
			query: `
			query X {
				a {
					a0(
						a0_0: {
							a0_00: 1.0
						}
						a0_1: "no"
					) {
						a00
					}
				}
				b(
					b_0: {
						b_00: "go"
					}
					b_1: [0.0, 1.0]
				) {
					b0
				}
				c(
					c_0: [
						{
							c_000: ["hohoho"]
						}
					]
					c_1: [
						[
							{
								c_1000: -1.0
								c_1001: [1.0, 0.0]
							}
						]
						[
							{
								c_1100: "hawk"
							}
							{
								c_1110: "falcon"
							}
						]
					]
				) {
					c0(
						c0_0: 0.0
					) {
						c00
					}
				}
			}
			`,
			expect: []pquery.QueryPart{
				{Hash("query.a.a0.a0_0.a0_00"), 1.0},
				{Hash("query.a.a0.a0_1"), []byte("no")},
				{Hash("query.a.a0.a00"), nil},
				{Hash("query.b.b_0.b_00"), []byte("go")},
				{Hash("query.b.b_1"), &[]any{0.0, 1.0}},
				{Hash("query.b.b0"), nil},
				{
					Hash("query.c.c_0"),
					&[]any{
						MakeMap(
							hamap.Pair[string, any]{
								Key: "c_000",
								Value: &[]any{
									[]byte("hohoho"),
								},
							},
						),
					},
				},
				{
					Hash("query.c.c_1"),
					&[]any{
						&[]any{
							MakeMap(
								hamap.Pair[string, any]{
									Key:   "c_1000",
									Value: -1.0,
								},
								hamap.Pair[string, any]{
									Key:   "c_1001",
									Value: &[]any{1.0, 0.0},
								},
							),
						},
						&[]any{
							MakeMap(
								hamap.Pair[string, any]{
									Key:   "c_1100",
									Value: []byte("hawk"),
								},
							),
							MakeMap(
								hamap.Pair[string, any]{
									Key:   "c_1110",
									Value: []byte("falcon"),
								},
							),
						},
					},
				},
				{Hash("query.c.c0.c0_0"), 0.0},
				{Hash("query.c.c0.c00"), nil},
			},
		},
		{
			operationName: "X",
			query: `
			mutation X {
				a {
					a0
				}
				b(
					b_0: 0.0
				) {
					b0
				}
			}
			`,
			expect: []pquery.QueryPart{
				{Hash("mutation.a.a0"), nil},
				{Hash("mutation.b.b_0"), 0.0},
				{Hash("mutation.b.b0"), nil},
			},
		},
	} {
		t.Run("", func(t *testing.T) {
			var i int

			gqlparse.NewParser().Parse(
				[]byte(td.query),
				[]byte(td.operationName),
				[]byte(td.variablesJSON),
				func(
					varValues [][]gqlparse.Token,
					operation []gqlparse.Token,
					selectionSet []gqlparse.Token,
				) {
					pquery.NewMaker(0).ParseQuery(
						varValues,
						operation[0].ID,
						selectionSet,
						func(qp pquery.QueryPart) (stop bool) {
							require.Equal(t, td.expect[i], qp)
							i++
							return false
						},
					)
				},
				func(err error) {
					t.Fatalf("unexpected parser error: %v", err)
				},
			)
		})
	}
}

func TestPrint(t *testing.T) {
	for _, td := range []struct {
		query         string
		operationName string
		variablesJSON string
		expect        string
	}{
		{
			operationName: "X",
			query: `
			query X {
				a {
					a0(
						a0_0: {
							a0_00: 1
						}
					)
				}
			}
			`,
			expect: fmt.Sprintf(`%d: 1
`, Hash("query.a.a0.a0_0.a0_00")),
		},
		{
			query: `
			query {
				a(
					a_0: [ 1, 2 ]
				)
			}
			`,
			expect: fmt.Sprintf(`%d:
  -:
    1
  -:
    2
`, Hash("query.a.a_0")),
		},
		{
			query: `
			query {
				a(
					a_0: [
						{
							a_000: 5
						}
						{
							a_010: [ 0, 1 ]
						}
					]
				)
			}
			`,
			expect: fmt.Sprintf(`%d:
  -:
    a_000:
      5
  -:
    a_010:
      -:
        0
      -:
        1
`, Hash("query.a.a_0")),
		},
	} {
		t.Run("", func(t *testing.T) {
			gqlparse.NewParser().Parse(
				[]byte(td.query),
				[]byte(td.operationName),
				[]byte(td.variablesJSON),
				func(
					varValues [][]gqlparse.Token,
					operation []gqlparse.Token,
					selectionSet []gqlparse.Token,
				) {
					b := new(bytes.Buffer)
					pquery.NewMaker(0).ParseQuery(
						varValues,
						operation[0].ID,
						selectionSet,
						func(qp pquery.QueryPart) (stop bool) {
							qp.Print(b)
							return false
						},
					)
					require.Equal(t, td.expect, b.String())
				},
				func(err error) {
					t.Fatalf("unexpected parser error: %v", err)
				},
			)
		})
	}
}

func Hash(s string) uint64 {
	h := xxhash.New(0)
	xxhash.Write(&h, s)
	return h.Sum64()
}

func MakeMap(items ...hamap.Pair[string, any]) *hamap.Map[string, any] {
	m := hamap.New[string, any](len(items), nil)
	for i := range items {
		m.Set(items[i].Key, items[i].Value)
	}
	return m
}
