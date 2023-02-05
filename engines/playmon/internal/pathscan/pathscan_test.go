package pathscan_test

import (
	"testing"

	"github.com/graph-guard/ggproxy/engines/playmon/internal/pathscan"
	"github.com/graph-guard/ggproxy/gqlparse"
	"github.com/stretchr/testify/require"
)

func TestPathScan(t *testing.T) {
	for _, tt := range []struct {
		Name          string
		Input         string
		ExpectedPaths []string
	}{
		{
			Name: "query",
			Input: `query {
				foo {
					bar {
						burr(x:4)
					}
					baz {
						... on Kraz {
							fraz
							graz(argument:{i:"foo",i2:"bar"}) {
								lum
							}
						}
						buzz(b:5, a:null, c:true)
						... on Guz {
							guz
						}
					}
				}
				mazz
			}`,
			ExpectedPaths: []string{
				"Q.foo.bar.burr|x",
				"Q.foo.baz&Kraz.fraz",
				"Q.foo.baz&Kraz.graz|argument",
				"Q.foo.baz&Kraz.graz.lum",
				"Q.foo.baz.buzz|b",
				"Q.foo.baz.buzz|a",
				"Q.foo.baz.buzz|c",
				"Q.foo.baz&Guz.guz",
				"Q.mazz",
			},
		},
		{
			Name:  "mutation_with_vars",
			Input: `mutation($baz:String="baz"){foo(bar:$baz)}`,
			ExpectedPaths: []string{
				"M.foo|bar",
			},
		},
		{
			Name:  "subscription",
			Input: `subscription{foo}`,
			ExpectedPaths: []string{
				"S.foo",
			},
		},
	} {
		t.Run(tt.Name, func(t *testing.T) {
			ps := pathscan.New(0, 0)
			p := gqlparse.NewParser()
			var actualPaths []string
			p.Parse(
				[]byte(tt.Input), nil, nil,
				func(
					varValues [][]gqlparse.Token,
					operation, selectionSet []gqlparse.Token,
				) {
					ps.Scan(operation, func(b []byte) (stop bool) {
						actualPaths = append(actualPaths, string(b))
						return false
					})
				},
				func(err error) {
					t.Fatalf("unexpected GraphQL parsing error: %v", err)
				},
			)
			require.Equal(t, tt.ExpectedPaths, actualPaths)
		})
	}
}
