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

var testsInTokens = []struct {
	Name               string
	GraphQLOperation   string
	VariablePaths      []string
	ExpectedStructural []string
	ExpectedVarVal     map[string]int // path->index
	ExpectedArg        map[string]int // path->index
}{
	{
		Name: "query_selections_noargs",
		GraphQLOperation: `query {
				foo { foo2 }
				bar { bar2 bar3 bar4 }
				bazz
				fuzz
				maz { kraz { glaz { traz } } }
				jazz
			}`,
		ExpectedStructural: []string{
			"Q.foo.foo2",
			"Q.bar.bar2",
			"Q.bar.bar3",
			"Q.bar.bar4",
			"Q.bazz",
			"Q.fuzz",
			"Q.maz.kraz.glaz.traz",
			"Q.jazz",
		},
	},
	{
		Name: "query_type_conditions",
		GraphQLOperation: `query {
				u {
					... on Foo {
						foo foo2
						... on Far {
							far
							far2
						}
					}
					... on Bar { bar bar2 }
				}
			}`,
		ExpectedStructural: []string{
			"Q.u&Foo.foo",
			"Q.u&Foo.foo2",
			"Q.u&Foo&Far.far",
			"Q.u&Foo&Far.far2",
			"Q.u&Bar.bar",
			"Q.u&Bar.bar2",
		},
	},
	{
		Name:               "subscription",
		GraphQLOperation:   `subscription { s }`,
		ExpectedStructural: []string{"S.s"},
	},
	{
		Name:               "mutation",
		GraphQLOperation:   `mutation { m }`,
		ExpectedStructural: []string{"M.m"},
	},
	{
		Name: "args",
		GraphQLOperation: `query{
			titles(options:{lang: DE})
			entities(filter:["filter","this","out"])
		}`,
		VariablePaths: []string{
			"Q.titles|options",
			"Q.entities|filter",
		},
		ExpectedStructural: []string{
			"Q.titles|options,",
			"Q.entities|filter,",
		},
		ExpectedArg: map[string]int{
			"Q.titles|options":  3,
			"Q.entities|filter": 11,
		},
		ExpectedVarVal: map[string]int{
			"Q.titles|options":  4,
			"Q.entities|filter": 12,
		},
	},
	{
		Name: "args_complex",
		GraphQLOperation: `mutation($variable:Int! = 42){
				foo(i:42, t:true, f:false, n:null)
				maz(var:$variable) {
					kraz(x:"""text""") {
						fraz(x:"more text") {
							graz
						}
					}
				}
				bazz(object:{array:[1,2,3,null], enum: ENUM_VALUE})
				bar(strings:["array","of","strings"])
			}`,
		VariablePaths: []string{
			"M.foo|i",
			"M.foo|t",
			"M.foo|f",
			"M.foo|n",
			"M.maz|var",
			"M.maz.kraz|x",
			"M.maz.kraz.fraz|x",
			"M.bazz|object",
			"M.bar|strings",
		},
		ExpectedStructural: []string{
			"M.foo|f,i,n,t,",
			"M.maz|var,.kraz|x,.fraz|x,.graz",
			"M.bazz|object,",
			"M.bar|strings,",
		},
		ExpectedArg: map[string]int{
			"M.foo|i":           3,
			"M.foo|t":           5,
			"M.foo|f":           7,
			"M.foo|n":           9,
			"M.maz|var":         14,
			"M.maz.kraz|x":      20,
			"M.maz.kraz.fraz|x": 26,
			"M.bazz|object":     36,
			"M.bar|strings":     51,
		},
		ExpectedVarVal: map[string]int{
			"M.foo|i":           4,
			"M.foo|t":           6,
			"M.foo|f":           8,
			"M.foo|n":           10,
			"M.maz|var":         15,
			"M.maz.kraz|x":      21,
			"M.maz.kraz.fraz|x": 27,
			"M.bazz|object":     37,
			"M.bar|strings":     52,
		},
	},
	{
		Name: "query_complex",
		GraphQLOperation: `query {
				foo {
					bar {
						burr(x:4)
					}
					baz {
						... on Kraz {
							fraz
							graz(argument:{i:"foo",i2:"bar"}) {
								lum
								klum
							}
						}
						buzz(b:5, a:null, c:true)
						brazz(b:5, a:null, c:true)
						... on Guz {
							guz
							guzz {
								blaz
							}
						}
					}
				}
				mazz
				laz(x:ENUM_VALUE)
			}`,
		ExpectedStructural: []string{
			"Q.foo.bar.burr|x,",
			"Q.foo.baz&Kraz.fraz",
			"Q.foo.baz&Kraz.graz|argument,.lum",
			"Q.foo.baz&Kraz.graz|argument,.klum",
			"Q.foo.baz.buzz|a,b,c,",
			"Q.foo.baz.brazz|a,b,c,",
			"Q.foo.baz&Guz.guz",
			"Q.foo.baz&Guz.guzz.blaz",
			"Q.mazz",
			"Q.laz|x,",
		},
		ExpectedArg: map[string]int{
			"Q.foo.bar.burr|x":             7,
			"Q.foo.baz&Kraz.graz|argument": 18,
			"Q.foo.baz.buzz|b":             33,
			"Q.foo.baz.buzz|a":             35,
			"Q.foo.baz.buzz|c":             37,
			"Q.foo.baz.brazz|b":            42,
			"Q.foo.baz.brazz|a":            44,
			"Q.foo.baz.brazz|c":            46,
			"Q.laz|x":                      62,
		},
	},
	{
		Name: "query_gqtvar_object",
		GraphQLOperation: `query {
			f(obj:{foo:{bar:1,baz:2}, fraz:3})
		}`,
		VariablePaths: []string{
			"Q.f|obj",
			"Q.f|obj/foo",
			"Q.f|obj/foo/bar",
			"Q.f|obj/foo/baz",
			"Q.f|obj/fraz",
		},
		ExpectedStructural: []string{
			"Q.f|obj,",
		},
		ExpectedArg: map[string]int{
			"Q.f|obj": 3,
		},
		ExpectedVarVal: map[string]int{
			"Q.f|obj":         4,
			"Q.f|obj/foo":     6,
			"Q.f|obj/foo/bar": 8,
			"Q.f|obj/foo/baz": 10,
			"Q.f|obj/fraz":    13,
		},
	},
	{
		Name: "query_gqtvar_object_partialvars",
		GraphQLOperation: `query {
			f(obj:{foo:{bar:1,baz:2}, fraz:3})
		}`,
		VariablePaths: []string{
			"Q.f|obj/foo/baz",
			"Q.f|obj/fraz",
		},
		ExpectedStructural: []string{
			"Q.f|obj,",
		},
		ExpectedArg: map[string]int{
			"Q.f|obj": 3,
		},
		ExpectedVarVal: map[string]int{
			"Q.f|obj/foo/baz": 10,
			"Q.f|obj/fraz":    13,
		},
	},
}

func TestInTokens(t *testing.T) {
	for _, tt := range testsInTokens {
		t.Run(tt.Name, func(t *testing.T) {
			ps := pathscan.New(0, 0)
			p := gqlparse.NewParser()
			actualStructuralPaths := []string{}
			actualArgPaths := make(map[string]int)
			actualVarPaths := make(map[string]int)
			variablePaths := make(map[string]int, len(tt.VariablePaths))
			for _, p := range tt.VariablePaths {
				variablePaths[p] = 0
			}
			p.Parse(
				[]byte(tt.GraphQLOperation), nil, nil,
				func(
					varValues [][]gqlparse.Token,
					operation, selectionSet []gqlparse.Token,
				) {
					ps.InTokens(
						operation[0].ID,
						selectionSet,
						variablePaths,
						func(path []byte) (stop bool) { // On structural
							actualStructuralPaths = append(
								actualStructuralPaths, string(path),
							)
							return false
						},
						func(path []byte, i int) (stop bool) { // On argument
							actualArgPaths[string(path)] = i
							return false
						},
						func(path []byte, i int) (stop bool) { // On GQT variable
							actualVarPaths[string(path)] = i
							return false
						},
					)
				},
				func(err error) {
					t.Fatalf("unexpected GraphQL parsing error: %v", err)
				},
			)
			require.Equal(
				t, tt.ExpectedStructural, actualStructuralPaths,
				"structural paths",
			)
			if tt.ExpectedVarVal == nil {
				tt.ExpectedVarVal = map[string]int{}
			}
			if tt.ExpectedArg == nil {
				tt.ExpectedArg = map[string]int{}
			}
			require.Equal(t, tt.ExpectedArg, actualArgPaths, "argument paths")
			require.Equal(t, tt.ExpectedVarVal, actualVarPaths, "variable paths")
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
			map[string]int{
				// No variable paths
			},
			func(path []byte) (stop bool) { // On structural
				t.Fatal("this function isn't expected to be called")
				return false
			},
			func(path []byte, i int) (stop bool) { // On argument
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

var testsInAST = []struct {
	Name             string
	GQTTemplateSrc   string
	ExpectStructural []string
	ExpectedArgPaths []string
	ExpectedVarPaths []string
}{
	{
		Name:           "subscription_single",
		GQTTemplateSrc: `subscription{foo}`,
		ExpectStructural: []string{
			"S.foo",
		},
	},
	{
		Name:           "subscription_multiple",
		GQTTemplateSrc: `subscription{foo bar}`,
		ExpectStructural: []string{
			"S.foo",
			"S.bar",
		},
	},
	{
		Name:           "args",
		GQTTemplateSrc: `query{foo(b:*,a:"t") bar(c:42)}`,
		ExpectStructural: []string{
			"Q.foo|a,b,",
			"Q.bar|c,",
		},
		ExpectedArgPaths: []string{
			"Q.foo|b",
			"Q.foo|a",
			"Q.bar|c",
		},
	},
	{
		Name:           "args_with_subselections",
		GQTTemplateSrc: `query{foo(b:*,a:"t") { fraz kraz(c:*) }}`,
		ExpectStructural: []string{
			"Q.foo|a,b,.fraz",
			"Q.foo|a,b,.kraz|c,",
		},
		ExpectedArgPaths: []string{
			"Q.foo|b",
			"Q.foo|a",
			"Q.foo.kraz|c",
		},
	},
	{
		Name:           "mutation_with_vars",
		GQTTemplateSrc: `mutation{foo(bar=$bar:{x:*,y:*})}`,
		ExpectStructural: []string{
			"M.foo|bar,",
		},
		ExpectedArgPaths: []string{
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
		ExpectStructural: []string{
			"M.foo|bar,.fa2",
			"M.foo|bar,.fo2|a,b,.fo3|c,",
			"M.foo|bar,.fo2|a,b,.fa3",
			"M.bazz|x,",
		},
		ExpectedArgPaths: []string{
			"M.foo|bar",
			"M.foo.fo2|b",
			"M.foo.fo2|a",
			"M.foo.fo2.fo3|c",
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
		ExpectStructural: []string{
			"M.foo.bar|bar1,bar2,.baz|o,o2,x,y,",
		},
		ExpectedArgPaths: []string{
			"M.foo.bar|bar1",
			"M.foo.bar|bar2",
			"M.foo.bar.baz|o",
			"M.foo.bar.baz|o2",
			"M.foo.bar.baz|x",
			"M.foo.bar.baz|y",
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
		ExpectStructural: []string{
			"Q.u|u1,u2,&Foo.foo1|x,y,",
			"Q.u|u1,u2,&Foo.foo2|x,",
			"Q.u|u1,u2,&Bar.bar1|x,",
			"Q.u|u1,u2,&Bar.bar2|x,y,",
		},
		ExpectedArgPaths: []string{
			"Q.u|u2",
			"Q.u|u1",
			"Q.u&Foo.foo1|x",
			"Q.u&Foo.foo1|y",
			"Q.u&Foo.foo2|x",
			"Q.u&Bar.bar1|x",
			"Q.u&Bar.bar2|x",
			"Q.u&Bar.bar2|y",
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
		ExpectStructural: []string{
			"Q.foo.bar.burr|x,",
			"Q.foo.baz&Kraz.fraz",
			"Q.foo.baz&Kraz.graz|argument,.lum",
			"Q.foo.baz&Guz.guz",
			"Q.foo.baz.buzz|a,b,c,",
			"Q.mazz",
		},
		ExpectedArgPaths: []string{
			"Q.foo.bar.burr|x",
			"Q.foo.baz&Kraz.graz|argument",
			"Q.foo.baz.buzz|b",
			"Q.foo.baz.buzz|a",
			"Q.foo.baz.buzz|c",
		},
	},
}

func TestInAst(t *testing.T) {
	for _, tt := range testsInAST {
		t.Run(tt.Name, func(t *testing.T) {
			o, _, errs := gqt.Parse([]byte(tt.GQTTemplateSrc))
			require.Nil(t, errs)

			var actualPaths, actualArgPaths, actualVarPaths []string
			pathscan.InAST(o, func(path []byte, e gqt.Expression) (stop bool) {
				// On structural
				actualPaths = append(actualPaths, string(path))
				require.NotNil(t, e)
				return false
			}, func(path []byte, e gqt.Expression) (stop bool) {
				// On argument
				actualArgPaths = append(actualArgPaths, string(path))
				return false
			}, func(path []byte, e gqt.Expression) (stop bool) {
				// On variable
				actualVarPaths = append(actualVarPaths, string(path))
				return false
			})

			sort.Strings(tt.ExpectStructural)
			sort.Strings(actualPaths)
			require.Equal(t, tt.ExpectStructural, actualPaths, "structural paths")

			sort.Strings(tt.ExpectedArgPaths)
			sort.Strings(actualArgPaths)
			require.Equal(t, tt.ExpectedArgPaths, actualArgPaths, "argument paths")

			sort.Strings(tt.ExpectedVarPaths)
			sort.Strings(actualVarPaths)
			require.Equal(t, tt.ExpectedVarPaths, actualVarPaths, "variable paths")
		})
	}
}
