package pathscan_test

import (
	"sort"
	"testing"

	"github.com/graph-guard/ggproxy/engine/playmon/internal/pathscan"
	"github.com/graph-guard/ggproxy/gqlparse"
	"github.com/graph-guard/gqlscan"
	"github.com/graph-guard/gqt/v4"
	"github.com/stretchr/testify/require"
)

func TestInTokens(t *testing.T) {
	for _, tt := range []struct {
		Name          string
		Input         string
		VariablePaths map[string]struct{}
		Expected      map[string]int // path->index
		ExpectedVar   map[string]int // path->index
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
			Expected: map[string]int{
				"Q.foo.bar.burr|x":             7,
				"Q.foo.baz&Kraz.fraz":          15,
				"Q.foo.baz&Kraz.graz|argument": 18,
				"Q.foo.baz&Kraz.graz.lum":      27,
				"Q.foo.baz.buzz|b":             32,
				"Q.foo.baz.buzz|a":             34,
				"Q.foo.baz.buzz|c":             36,
				"Q.foo.baz&Guz.guz":            41,
				"Q.mazz":                       45,
			},
		},
		{
			Name:  "mutation_with_vars",
			Input: `mutation($baz:String="baz"){foo(bar:$baz)}`,
			Expected: map[string]int{
				"M.foo|bar": 3,
			},
		},
		{
			Name:  "subscription",
			Input: `subscription{foo}`,
			Expected: map[string]int{
				"S.foo": 1,
			},
		},
	} {
		t.Run(tt.Name, func(t *testing.T) {
			ps := pathscan.New(0, 0)
			p := gqlparse.NewParser()
			actualPaths := make(map[string]int)
			actualVarPaths := make(map[string]int)
			p.Parse(
				[]byte(tt.Input), nil, nil,
				func(
					varValues [][]gqlparse.Token,
					operation, selectionSet []gqlparse.Token,
				) {
					ps.InTokens(
						operation[0].ID,
						selectionSet,
						tt.VariablePaths,
						func(b []byte, i int) (stop bool) { // On structural
							actualPaths[string(b)] = i
							return false
						},
						func(b []byte, i int) (stop bool) { // On variable
							actualVarPaths[string(b)] = i
							return false
						},
					)
				},
				func(err error) {
					t.Fatalf("unexpected GraphQL parsing error: %v", err)
				},
			)
			require.Equal(t, tt.Expected, actualPaths)
			if tt.ExpectedVar == nil {
				tt.ExpectedVar = map[string]int{}
			}
			require.Equal(t, tt.ExpectedVar, actualVarPaths)
		})
	}
}

func TestInTokensPanic(t *testing.T) {
	ps := pathscan.New(0, 0)
	require.Panics(t, func() {
		ps.InTokens(
			gqlscan.TokenOprName,
			[]gqlparse.Token{
				{ID: gqlscan.TokenSet},
				{ID: gqlscan.TokenField, Value: []byte("foo")},
				{ID: gqlscan.TokenSetEnd},
			},
			map[string]struct{}{
				// No variable paths
			},
			func(path []byte, i int) (stop bool) { // On structural
				t.Fatal("this function isn't expected to be called")
				return false
			},
			func(path []byte, i int) (stop bool) { // On variable
				t.Fatal("this function isn't expected to be called")
				return false
			},
		)
	})
}

func TestInAst(t *testing.T) {
	for _, tt := range []struct {
		Name             string
		GQTTemplateSrc   string
		ExpectedPaths    []string
		ExpectedVarPaths []string
	}{
		{
			Name:           "subscription_single",
			GQTTemplateSrc: `subscription{foo}`,
			ExpectedPaths: []string{
				"S.foo",
			},
		},
		{
			Name:           "subscription_multiple",
			GQTTemplateSrc: `subscription{foo bar}`,
			ExpectedPaths: []string{
				"S.foo",
				"S.bar",
			},
		},
		{
			Name:           "args",
			GQTTemplateSrc: `query{foo(b:*,a:"t") bar(c:42)}`,
			ExpectedPaths: []string{
				"Q.bar|c",
				"Q.foo|a,b",
			},
		},
		{
			Name:           "args_with_subselections",
			GQTTemplateSrc: `query{foo(b:*,a:"t") { fraz kraz(c:*) }}`,
			ExpectedPaths: []string{
				"Q.foo|a,b.fraz",
				"Q.foo|a,b.kraz|c",
			},
		},
		{
			Name:           "mutation_with_vars",
			GQTTemplateSrc: `mutation{foo(bar=$bar:{x:*,y:*})}`,
			ExpectedPaths: []string{
				"M.foo|bar",
			},
			ExpectedVarPaths: []string{
				"M.foo|bar",
			},
		},
		{
			Name: "mutation_with_vars_on_multiple_levels",
			GQTTemplateSrc: `mutation{
				foo(bar=$bar:*){
					fo2(b=$b:*,a=$a:*){
						fo3(c=$c:*)
						fa3
					}
					fa2
				}
				bazz(x:*)
			}`,
			ExpectedPaths: []string{
				"M.foo|bar.fa2",
				"M.foo|bar.fo2|a,b.fo3|c",
				"M.foo|bar.fo2|a,b.fa3",
				"M.bazz|x",
			},
			ExpectedVarPaths: []string{
				"M.foo|bar",
				"M.foo.fo2|b",
				"M.foo.fo2|a",
				"M.foo.fo2.fo3|c",
			},
		},
		{
			Name: "mutation_var_in_obj",
			GQTTemplateSrc: `mutation{
				foo {
					bar(bar1:*,bar2:*) {
						baz(
							o: {
								so=$so: {
									x=$x: *,
									y=$y: *,
								}
							},
							o2: $so,
							x=$x2: $x,
							y: $x2 || $y,
						)
					}
				}
			}`,
			ExpectedPaths: []string{
				"M.foo.bar|bar1,bar2.baz|o,o2,x,y",
			},
			ExpectedVarPaths: []string{
				"M.foo.bar.baz|o/so",
				"M.foo.bar.baz|o/so/x",
				"M.foo.bar.baz|o/so/y",
				"M.foo.bar.baz|x",
			},
		},
		{
			Name: "type_condition",
			GQTTemplateSrc: `query{
				u(u2:*,u1:*) {
					... on Foo {
						foo1(x=$foo1x:*, y:*)
						foo2(x:*)
					}
					... on Bar {
						bar1(x:*)
						bar2(x=$bar2x:*, y:*)
					}
				}
			}`,
			ExpectedPaths: []string{
				"Q.u|u1,u2&Foo.foo1|x,y",
				"Q.u|u1,u2&Foo.foo2|x",
				"Q.u|u1,u2&Bar.bar1|x",
				"Q.u|u1,u2&Bar.bar2|x,y",
			},
			ExpectedVarPaths: []string{
				"Q.u&Foo.foo1|x",
				"Q.u&Bar.bar2|x",
			},
		},
		{
			Name: "complex",
			GQTTemplateSrc: `query {
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
				"Q.foo.baz&Kraz.graz|argument.lum",
				"Q.foo.baz&Guz.guz",
				"Q.foo.baz.buzz|a,b,c",
				"Q.mazz",
			},
		},
	} {
		t.Run(tt.Name, func(t *testing.T) {
			o, _, errs := gqt.Parse([]byte(tt.GQTTemplateSrc))
			require.Nil(t, errs)
			var actualPaths, actualVarPaths []string
			pathscan.InAST(o, func(path []byte, e gqt.Expression) (stop bool) {
				actualPaths = append(actualPaths, string(path))
				require.NotNil(t, e)
				return false
			}, func(path []byte, e gqt.Expression) (stop bool) {
				actualVarPaths = append(actualVarPaths, string(path))
				return false
			})
			sort.Strings(tt.ExpectedPaths)
			sort.Strings(actualPaths)
			require.Equal(t, tt.ExpectedPaths, actualPaths, "structural paths")

			sort.Strings(tt.ExpectedVarPaths)
			sort.Strings(actualVarPaths)
			require.Equal(t, tt.ExpectedVarPaths, actualVarPaths, "variable paths")
		})
	}
}
