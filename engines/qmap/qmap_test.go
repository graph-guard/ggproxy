package qmap_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/graph-guard/ggproxy/engines/qmap"
	"github.com/graph-guard/ggproxy/gqlreduce"
	"github.com/graph-guard/ggproxy/utilities/container/hamap"
	"github.com/graph-guard/ggproxy/utilities/xxhash"
	"github.com/graph-guard/gqlscan"
	"github.com/stretchr/testify/require"
)

func TestNewQueryMap(t *testing.T) {
	for _, td := range []struct {
		query  string
		expect qmap.QueryMap
	}{
		{
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
			expect: qmap.QueryMap{
				Hash("query.c.c0.c00"):        nil,
				Hash("query.a.a0.a00"):        nil,
				Hash("query.b.b0"):            nil,
				Hash("query.c.c0.c0_0"):       0.0,
				Hash("query.a.a0.a0_0.a0_00"): 1.0,
				Hash("query.a.a0.a0_1"):       []byte("no"),
				Hash("query.b.b_0.b_00"):      []byte("go"),
				Hash("query.b.b_1"):           &[]any{0.0, 1.0},
				Hash("query.c.c_0"): &[]any{
					MakeMap(
						hamap.Pair[string, any]{
							Key: "c_000",
							Value: &[]any{
								[]byte("hohoho"),
							},
						},
					),
				},
				Hash("query.c.c_1"): &[]any{
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
		},
		{
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
			expect: qmap.QueryMap{
				Hash("mutation.a.a0"):  nil,
				Hash("mutation.b.b0"):  nil,
				Hash("mutation.b.b_0"): 0.0,
			},
		},
	} {
		t.Run("", func(t *testing.T) {
			var tokens []gqlreduce.Token
			err := gqlscan.ScanAll([]byte(td.query), func(i *gqlscan.Iterator) {
				tokens = append(tokens, gqlreduce.Token{
					Type:  i.Token(),
					Value: i.Value(),
				})
			})
			require.Equal(t, false, err.IsErr())

			qmap.NewMaker(0).ParseQuery(tokens, func(qm qmap.QueryMap) {
				require.Equal(t, td.expect, qm)
			})
		})
	}
}

func TestPrint(t *testing.T) {
	for _, td := range []struct {
		query  string
		expect string
	}{
		{
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
			var tokens []gqlreduce.Token
			err := gqlscan.ScanAll([]byte(td.query), func(i *gqlscan.Iterator) {
				tokens = append(tokens, gqlreduce.Token{
					Type:  i.Token(),
					Value: i.Value(),
				})
			})
			require.Equal(t, false, err.IsErr())

			b := new(bytes.Buffer)
			qmap.NewMaker(0).ParseQuery(tokens, func(qm qmap.QueryMap) {
				qm.Print(b)
			})

			require.Equal(t, td.expect, b.String())
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
