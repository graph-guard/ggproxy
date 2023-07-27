package engine_test

import (
	"embed"
	_ "embed"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"

	"github.com/graph-guard/ggproxy/pkg/config"
	"github.com/graph-guard/ggproxy/pkg/engine/playmon"
	"github.com/graph-guard/ggproxy/pkg/gqlparse"
	"github.com/graph-guard/ggproxy/pkg/testsetup"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

//go:embed tests
var testsFS embed.FS

func TestPlaymonMatch(t *testing.T) {
	testSets, err := testsFS.ReadDir("tests")
	require.NoError(t, err)
	for _, d := range testSets {
		t.Run(d.Name(), func(t *testing.T) {
			if !d.IsDir() {
				t.Skip("not a directory")
			}
			setup, ok := testsetup.ByName(d.Name())
			if !ok {
				t.Fatalf("unknown test setup: %q", d.Name())
			}

			engine := playmon.New(setup.Config.ServicesEnabled[0])

			tests, err := fs.ReadDir(testsFS, filepath.Join("tests", d.Name()))
			require.NoError(t, err)
			for _, f := range tests {
				name := strings.TrimSuffix(f.Name(), ".yaml")
				t.Run(name, func(t *testing.T) {
					if !strings.HasSuffix(f.Name(), ".yaml") {
						t.Skip("missing '.yaml' extension")
					}
					if f.IsDir() {
						t.Skip("directory")
					}
					c, err := fs.ReadFile(testsFS, filepath.Join("tests", d.Name(), f.Name()))
					require.NoError(t, err)
					var ts Test
					err = yaml.Unmarshal(c, &ts)
					require.NoError(t, err)
					var errMsg string
					var matches []string

					var operationName []byte
					if ts.OperationName != "" {
						operationName = []byte(ts.OperationName)
					}
					var variablesJSON []byte
					if ts.VariablesJSON != "" {
						variablesJSON = []byte(ts.VariablesJSON)
					}

					engine.Match(
						[]byte(ts.Query), operationName, variablesJSON,
						func(operation, selectionSet []gqlparse.Token) (stop bool) {
							return false
						},
						func(template *config.Template) (stop bool) {
							matches = append(matches, template.ID)
							return false
						}, func(err error) {
							errMsg = err.Error()
						},
					)
					require.Equal(t, ts.ExpectError, errMsg)
					require.Equal(t, ts.ExpectMatches, matches)
				})
			}
		})
	}
}

type Test struct {
	Query         string   `yaml:"query"`
	OperationName string   `yaml:"operation-name"`
	VariablesJSON string   `yaml:"variables-json"`
	ExpectError   string   `yaml:"expect-error"`
	ExpectMatches []string `yaml:"expect-matches"`
}

// type MatchTest struct {
// 	ID string
// 	*QueryModel
// 	Templates []*config.Template
// }

// func readTestAsset(
// 	filesystem fs.FS, path string,
// ) (
// 	query *QueryModel, templates []*config.Template,
// ) {
// 	test, err := fs.ReadDir(filesystem, path)
// 	if err != nil {
// 		panic(err)
// 	}

// 	for _, f := range test {
// 		if f.IsDir() {
// 			continue
// 		}
// 		fn := f.Name()
// 		fp := filepath.Join(path, f.Name())
// 		if strings.HasSuffix(fn, ".gqt") {
// 			id := strings.ToLower(fn[:len(fn)-len(filepath.Ext(fn))])
// 			src, err := filesystem.Open(fp)
// 			if err != nil {
// 				panic(err)
// 			}
// 			b, err := io.ReadAll(src)
// 			if err != nil {
// 				panic(err)
// 			}

// 			meta, template, err := metadata.Parse(b)
// 			if err != nil {
// 				panic(err)
// 			}
// 			doc, errParser := gqt.Parse(template)
// 			if errParser.IsErr() {
// 				panic(errParser)
// 			}

// 			templates = append(templates, &config.Template{
// 				ID:       id,
// 				Source:   template,
// 				Document: doc,
// 				Name:     meta.Name,
// 				Tags:     meta.Tags,
// 			})
// 		}
// 		if strings.HasSuffix(fn, ".yml") || strings.HasSuffix(fn, ".yaml") {
// 			src, err := filesystem.Open(fp)
// 			if err != nil {
// 				panic(err)
// 			}
// 			d := yaml.NewDecoder(src)
// 			d.KnownFields(true)
// 			err = d.Decode(&query)
// 			if err != nil {
// 				panic(err)
// 			}
// 		}
// 	}

// 	return
// }

// func readTestAssets(filesystem fs.FS, path, prefix string) (assets []*MatchTest) {
// 	root, err := fs.ReadDir(filesystem, path)
// 	if err != nil {
// 		panic(err)
// 	}
// 	for _, testDir := range root {
// 		if !testDir.IsDir() {
// 			continue
// 		}
// 		testDirName := testDir.Name()
// 		testDirPath := filepath.Join(path, testDirName)
// 		if !strings.HasPrefix(testDirName, prefix) {
// 			continue
// 		}

// 		query, templates := readTestAsset(filesystem, testDirPath)
// 		assets = append(assets, &MatchTest{
// 			ID:         testDirName,
// 			QueryModel: query,
// 			Templates:  templates,
// 		})
// 	}

// 	return
// }

// func TestMatchAllPartedQuery(t *testing.T) {
// 	for _, td := range readTestAssets(testsFS, "assets/testassets", "test_") {
// 		t.Run(td.ID, func(t *testing.T) {
// 			rules := make(map[string]gqt.Doc, len(td.Templates))
// 			for _, r := range td.Templates {
// 				rules[r.ID] = r.Document
// 			}

// 			p := gqlparse.NewParser()
// 			rm, _ := rmap.New(rules, 0)

// 			p.Parse(
// 				[]byte(td.Query),
// 				[]byte(td.OperationName),
// 				[]byte(td.Variables),
// 				func(
// 					varVals [][]gqlparse.Token,
// 					operation []gqlparse.Token,
// 					selectionSet []gqlparse.Token,
// 				) {
// 					actual := []string{}
// 					rm.MatchAll(
// 						varVals,
// 						operation[0].ID,
// 						selectionSet,
// 						func(id string) {
// 							actual = append(actual, id)
// 						},
// 					)
// 					require.Len(t, actual, len(td.Expect))
// 					for _, e := range td.Expect {
// 						require.Contains(t, actual, e)
// 					}
// 				},
// 				func(err error) {
// 					t.Fatalf("unexpected error: %v", err)
// 				},
// 			)
// 		})
// 	}
// }

// func TestPrintPartedQuery(t *testing.T) {
// 	for _, td := range []struct {
// 		template string
// 		expect   string
// 	}{
// 		{
// 			template: `
// 			query {
// 				a(
// 					a_0: val = 0
// 				)
// 			}
// 			`,
// 			expect: fmt.Sprintf(`%d:
//     ConstraintValEqual: 0
//       0
// `, Hash("query.a.a_0")),
// 		},
// 		{
// 			template: `
// 			query {
// 				a(
// 					a_0: val = "a"
// 				)
// 			}
// 			`,
// 			expect: fmt.Sprintf(`%d:
//     ConstraintValEqual: 0
//       a
// `, Hash("query.a.a_0")),
// 		},
// 		{
// 			template: `
// 			query {
// 				a(
// 					a_0: val = {
// 						a_00: val = [val = 1, val = 2]
// 					}
// 				)
// 			}
// 			`,
// 			expect: fmt.Sprintf(`%d:
//     ConstraintValEqual: 0
//       -:
//         ConstraintValEqual:
//           1
//       -:
//         ConstraintValEqual:
//           2
// `, Hash("query.a.a_0.a_00")),
// 		},
// 		{
// 			template: `
// 			query {
// 				a(
// 					a_0: val = [ ... val = [ ... val <= 0 ] ]
// 				)
// 			}
// 			`,
// 			expect: fmt.Sprintf(`%d:
//     ConstraintMap: 0
//       ConstraintMap:
//         ConstraintValLessOrEqual:
//           0
// `, Hash("query.a.a_0")),
// 		},
// 		{
// 			template: `
// 			query {
// 				a(
// 					a_0: val = [
// 						val = {
// 							a_000: val > 5
// 						}
// 						val = {
// 							a_010: val = [val = 0, val = 1]
// 						}
// 					]
// 				)
// 			}
// 			`,
// 			expect: fmt.Sprintf(`%d:
//     ConstraintValEqual: 0
//       -:
//         ConstraintValEqual:
//           a_000:
//             ConstraintValGreater:
//               5
//       -:
//         ConstraintValEqual:
//           a_010:
//             ConstraintValEqual:
//               -:
//                 ConstraintValEqual:
//                   0
//               -:
//                 ConstraintValEqual:
//                   1
// `, Hash("query.a.a_0")),
// 		},
// 	} {
// 		t.Run("", func(t *testing.T) {
// 			b := new(bytes.Buffer)

// 			rd, err := gqt.Parse([]byte(td.template))
// 			require.False(t, err.IsErr())
// 			rm, _ := rmap.New(map[string]gqt.Doc{
// 				"rd": rd,
// 			}, 0)
// 			rm.Print(b)

// 			require.Equal(t, td.expect, b.String())
// 		})
// 	}
// }

// func Hash(s string) uint64 {
// 	h := xxhash.New(0)
// 	xxhash.Write(&h, s)
// 	return h.Sum64()
// }

// func TestNewQueryPart(t *testing.T) {
// 	for _, td := range []struct {
// 		query         string
// 		operationName string
// 		variablesJSON string
// 		expect        []pquery.QueryPart
// 	}{
// 		{
// 			operationName: "X",
// 			query: `
// 			query X {
// 				a {
// 					a0(
// 						a0_0: {
// 							a0_00: 1.0
// 						}
// 						a0_1: "no"
// 					) {
// 						a00
// 					}
// 				}
// 				b(
// 					b_0: {
// 						b_00: "go"
// 					}
// 					b_1: [0.0, 1.0]
// 				) {
// 					b0
// 				}
// 				c(
// 					c_0: [
// 						{
// 							c_000: ["hohoho"]
// 						}
// 					]
// 					c_1: [
// 						[
// 							{
// 								c_1000: -1.0
// 								c_1001: [1.0, 0.0]
// 							}
// 						]
// 						[
// 							{
// 								c_1100: "hawk"
// 							}
// 							{
// 								c_1110: "falcon"
// 							}
// 						]
// 					]
// 				) {
// 					c0(
// 						c0_0: 0.0
// 					) {
// 						c00
// 					}
// 				}
// 			}
// 			`,
// 			expect: []pquery.QueryPart{
// 				{ArgLeafIdx: 0, Hash: Hash("query.a.a0.a0_0.a0_00"), Value: 1.0},
// 				{ArgLeafIdx: 1, Hash: Hash("query.a.a0.a0_1"), Value: []byte("no")},
// 				{ArgLeafIdx: -1, Hash: Hash("query.a.a0.a00"), Value: nil},
// 				{ArgLeafIdx: 0, Hash: Hash("query.b.b_0.b_00"), Value: []byte("go")},
// 				{ArgLeafIdx: 1, Hash: Hash("query.b.b_1"), Value: &[]any{0.0, 1.0}},
// 				{ArgLeafIdx: -1, Hash: Hash("query.b.b0"), Value: nil},
// 				{
// 					ArgLeafIdx: 0,
// 					Hash:       Hash("query.c.c_0"),
// 					Value: &[]any{
// 						MakeMap(
// 							hamap.Pair[string, any]{
// 								Key: "c_000",
// 								Value: &[]any{
// 									[]byte("hohoho"),
// 								},
// 							},
// 						),
// 					},
// 				},
// 				{
// 					ArgLeafIdx: 1,
// 					Hash:       Hash("query.c.c_1"),
// 					Value: &[]any{
// 						&[]any{
// 							MakeMap(
// 								hamap.Pair[string, any]{
// 									Key:   "c_1000",
// 									Value: -1.0,
// 								},
// 								hamap.Pair[string, any]{
// 									Key:   "c_1001",
// 									Value: &[]any{1.0, 0.0},
// 								},
// 							),
// 						},
// 						&[]any{
// 							MakeMap(
// 								hamap.Pair[string, any]{
// 									Key:   "c_1100",
// 									Value: []byte("hawk"),
// 								},
// 							),
// 							MakeMap(
// 								hamap.Pair[string, any]{
// 									Key:   "c_1110",
// 									Value: []byte("falcon"),
// 								},
// 							),
// 						},
// 					},
// 				},
// 				{ArgLeafIdx: 0, Hash: Hash("query.c.c0.c0_0"), Value: 0.0},
// 				{ArgLeafIdx: -1, Hash: Hash("query.c.c0.c00"), Value: nil},
// 			},
// 		},
// 		{
// 			operationName: "X",
// 			query: `
// 			mutation X {
// 				a {
// 					a0
// 				}
// 				b(
// 					b_0: 0.0
// 				) {
// 					b0
// 				}
// 			}
// 			`,
// 			expect: []pquery.QueryPart{
// 				{ArgLeafIdx: -1, Hash: Hash("mutation.a.a0"), Value: nil},
// 				{ArgLeafIdx: 0, Hash: Hash("mutation.b.b_0"), Value: 0.0},
// 				{ArgLeafIdx: -1, Hash: Hash("mutation.b.b0"), Value: nil},
// 			},
// 		},
// 	} {
// 		t.Run("", func(t *testing.T) {
// 			var i int

// 			gqlparse.NewParser().Parse(
// 				[]byte(td.query),
// 				[]byte(td.operationName),
// 				[]byte(td.variablesJSON),
// 				func(
// 					varValues [][]gqlparse.Token,
// 					operation []gqlparse.Token,
// 					selectionSet []gqlparse.Token,
// 				) {
// 					pquery.NewMaker(0).ParseQuery(
// 						varValues,
// 						operation[0].ID,
// 						selectionSet,
// 						func(qp pquery.QueryPart) (stop bool) {
// 							require.Equal(t, td.expect[i], qp)
// 							i++
// 							return false
// 						},
// 					)
// 				},
// 				func(err error) {
// 					t.Fatalf("unexpected parser error: %v", err)
// 				},
// 			)
// 		})
// 	}
// }

// func TestPrint(t *testing.T) {
// 	for _, td := range []struct {
// 		query         string
// 		operationName string
// 		variablesJSON string
// 		expect        string
// 	}{
// 		{
// 			operationName: "X",
// 			query: `
// 			query X {
// 				a {
// 					a0(
// 						a0_0: {
// 							a0_00: 1
// 						}
// 					)
// 				}
// 			}
// 			`,
// 			expect: fmt.Sprintf(`%d: 1
// `, Hash("query.a.a0.a0_0.a0_00")),
// 		},
// 		{
// 			query: `
// 			query {
// 				a(
// 					a_0: [ 1, 2 ]
// 				)
// 			}
// 			`,
// 			expect: fmt.Sprintf(`%d:
//   -:
//     1
//   -:
//     2
// `, Hash("query.a.a_0")),
// 		},
// 		{
// 			query: `
// 			query {
// 				a(
// 					a_0: [
// 						{
// 							a_000: 5
// 						}
// 						{
// 							a_010: [ 0, 1 ]
// 						}
// 					]
// 				)
// 			}
// 			`,
// 			expect: fmt.Sprintf(`%d:
//   -:
//     a_000:
//       5
//   -:
//     a_010:
//       -:
//         0
//       -:
//         1
// `, Hash("query.a.a_0")),
// 		},
// 	} {
// 		t.Run("", func(t *testing.T) {
// 			gqlparse.NewParser().Parse(
// 				[]byte(td.query),
// 				[]byte(td.operationName),
// 				[]byte(td.variablesJSON),
// 				func(
// 					varValues [][]gqlparse.Token,
// 					operation []gqlparse.Token,
// 					selectionSet []gqlparse.Token,
// 				) {
// 					b := new(bytes.Buffer)
// 					pquery.NewMaker(0).ParseQuery(
// 						varValues,
// 						operation[0].ID,
// 						selectionSet,
// 						func(qp pquery.QueryPart) (stop bool) {
// 							qp.Print(b)
// 							return false
// 						},
// 					)
// 					require.Equal(t, td.expect, b.String())
// 				},
// 				func(err error) {
// 					t.Fatalf("unexpected parser error: %v", err)
// 				},
// 			)
// 		})
// 	}
// }

// func MakeMap(items ...hamap.Pair[string, any]) *hamap.Map[string, any] {
// 	m := hamap.New[string, any](len(items), nil)
// 	for i := range items {
// 		m.Set(items[i].Key, items[i].Value)
// 	}
// 	return m
// }
