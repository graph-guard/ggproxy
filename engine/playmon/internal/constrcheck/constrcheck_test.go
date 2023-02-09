package constrcheck_test

// import (
// 	"embed"
// 	"io/fs"
// 	"sort"
// 	"strings"
// 	"testing"

// 	"github.com/graph-guard/ggproxy/engines/playmon/internal/constrcheck"
// 	"github.com/graph-guard/ggproxy/engines/playmon/internal/constrcheck/internal/test"
// 	"github.com/graph-guard/ggproxy/gqlparse"
// 	"github.com/graph-guard/gqt/v4"
// 	"github.com/stretchr/testify/require"
// 	"github.com/vektah/gqlparser/v2"
// 	gqlast "github.com/vektah/gqlparser/v2/ast"
// )

// //go:embed tests
// var testsFS embed.FS

// func TestCheck(t *testing.T) {
// 	d, err := fs.ReadDir(testsFS, "tests")
// 	require.NoError(t, err)

// 	for _, do := range d {
// 		fileName := do.Name()
// 		if do.IsDir() {
// 			t.Run(fileName, func(t *testing.T) {
// 				t.Skipf("ignoring directory %q", fileName)
// 			})
// 			continue
// 		}
// 		if !strings.HasSuffix(fileName, ".yml") {
// 			t.Run(fileName, func(t *testing.T) {
// 				t.Skipf("ignoring file %q", fileName)
// 			})
// 			continue
// 		}

// 		t.Run(strings.TrimSuffix(fileName, ".yml"), func(t *testing.T) {
// 			ts, err := test.Parse(testsFS, fileName)
// 			require.NoError(t, err)
// 			var withschema, schemaless *gqt.Operation
// 			var schemaAST *gqlast.Schema
// 			{
// 				p, err := gqt.NewParser([]gqt.Source{
// 					{Name: "schema.graphqls", Content: ts.Schema},
// 				})
// 				require.NoError(t, err, "unexpected error in schema")
// 				opr, _, errs := p.Parse([]byte(ts.Template))
// 				require.Len(t, errs, 0, "unexpected errors: %#v", errs)
// 				withschema = opr

// 				schemaAST, err = gqlparser.LoadSchema(&gqlast.Source{
// 					Name: "schema.graphqls", Input: ts.Schema,
// 				})
// 				require.NoError(t, err)
// 			}
// 			{
// 				opr, _, errs := gqt.Parse([]byte(ts.Template))
// 				require.Len(t, errs, 0, "unexpected errors: %#v", errs)
// 				schemaless = opr
// 			}

// 			run := func(t *testing.T, opr *gqt.Operation) {
// 				t.Helper()
// 				c := constrcheck.New(opr, schemaAST)

// 				paths := make([]string, 0, len(ts.Inputs))
// 				for path := range ts.Inputs {
// 					paths = append(paths, path)
// 				}
// 				sort.Strings(paths)

// 				variables := make([][]gqlparse.Token, len(ts.Inputs))
// 				for path, v := range ts.Inputs {
// 					inputs[path] = v.Tokens
// 				}

// 				c.Init(inputs)
// 				for _, path := range paths {
// 					if len(ts.Inputs[path].Tokens) < 1 {
// 						t.Errorf("missing test value for path %q", path)
// 						continue
// 					}

// 					if ts.Inputs[path].Match == nil {
// 						i := ts.Inputs[path]
// 						t := false
// 						i.Match = &t
// 						ts.Inputs[path] = i
// 					}
// 					if ts.Inputs[path].MatchSchemaless == nil {
// 						i := ts.Inputs[path]
// 						i.MatchSchemaless = i.Match
// 						ts.Inputs[path] = i
// 					}

// 					actual := c.Check(path)
// 					if opr.Def != nil {
// 						require.Equal(
// 							t, *ts.Inputs[path].Match, actual,
// 							"checking %q", path,
// 						)
// 					} else {
// 						require.Equal(
// 							t, *ts.Inputs[path].MatchSchemaless, actual,
// 							"checking %q", path,
// 						)
// 					}
// 				}
// 			}

// 			t.Run("schema", func(t *testing.T) { run(t, withschema) })
// 			t.Run("schemaless", func(t *testing.T) { run(t, schemaless) })
// 		})
// 	}
// }
