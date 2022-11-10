package engine_test

import (
	_ "embed"
	"testing"

	"github.com/graph-guard/ggproxy/engines/rmap"
	"github.com/graph-guard/gqt"
	"github.com/stretchr/testify/require"
)

func TestConstraintIdAndValue(t *testing.T) {
	for _, td := range []struct {
		input gqt.Expression
		id    rmap.Constraint
		value any
		err   error
	}{
		{
			input: &gqt.ConstrMap{
				Constraint: *new(gqt.Expression),
			},
			id:    rmap.ConstraintMap,
			value: *new(gqt.Expression),
		},
		{
			input: &gqt.ConstrAny{},
			id:    rmap.ConstraintAny,
			value: nil,
		},
		{
			input: &gqt.ConstrEquals{
				Value: &gqt.Object{
					Fields: []*gqt.ObjectField{
						{
							Name: gqt.Name{
								Name: "a",
							},
							Constraint: &gqt.ConstrLessOrEqual{
								Value: &gqt.Float{
									Value: 42.0,
								},
							},
						},
					},
				},
			},
			id: rmap.ConstraintValEqual,
			value: &gqt.Object{
				Fields: []*gqt.ObjectField{
					{
						Name: gqt.Name{
							Name: "a",
						},
						Constraint: &gqt.ConstrLessOrEqual{
							Value: &gqt.Float{
								Value: 42.0,
							},
						},
					},
				},
			},
		},
		{
			input: &gqt.ConstrGreater{
				Value: &gqt.Float{
					Value: 42.0,
				},
			},
			id: rmap.ConstraintValGreater,
			value: &gqt.Float{
				Value: 42.0,
			},
		},
		{
			input: &gqt.ConstrLess{
				Value: &gqt.Float{
					Value: 42.0,
				},
			},
			id: rmap.ConstraintValLess,
			value: &gqt.Float{
				Value: 42.0,
			},
		},
		{
			input: &gqt.ConstrGreaterOrEqual{
				Value: &gqt.Float{
					Value: 69.0,
				},
			},
			id: rmap.ConstraintValGreaterOrEqual,
			value: &gqt.Float{
				Value: 69.0,
			},
		},
		{
			input: &gqt.ConstrLessOrEqual{
				Value: &gqt.Float{
					Value: 69.0,
				},
			},
			id: rmap.ConstraintValLessOrEqual,
			value: &gqt.Float{
				Value: 69.0,
			},
		},
		{
			input: &gqt.ConstrLenEquals{
				Value: &gqt.Int{
					Value: 1984,
				},
			},
			id: rmap.ConstraintLenEqual,
			value: &gqt.Int{
				Value: 1984,
			},
		},
		{
			input: &gqt.ConstrLenNotEquals{
				Value: &gqt.Int{
					Value: 1984,
				},
			},
			id: rmap.ConstraintLenNotEqual,
			value: &gqt.Int{
				Value: 1984,
			},
		},
		{
			input: &gqt.ConstrLenGreater{
				Value: &gqt.Int{
					Value: 282,
				},
			},
			id: rmap.ConstraintLenGreater,
			value: &gqt.Int{
				Value: 282,
			},
		},
		{
			input: &gqt.ConstrLenLess{
				Value: &gqt.Int{
					Value: 282,
				},
			},
			id: rmap.ConstraintLenLess,
			value: &gqt.Int{
				Value: 282,
			},
		},
		{
			input: &gqt.ConstrLenGreaterOrEqual{
				Value: &gqt.Int{
					Value: 27015,
				},
			},
			id: rmap.ConstraintLenGreaterOrEqual,
			value: &gqt.Int{
				Value: 27015,
			},
		},
		{
			input: &gqt.ConstrLenLessOrEqual{
				Value: &gqt.Int{
					Value: 27015,
				},
			},
			id: rmap.ConstraintLenLessOrEqual,
			value: &gqt.Int{
				Value: 27015,
			},
		},
	} {
		t.Run("", func(t *testing.T) {
			id, value := rmap.ConstraintIdAndValue(td.input)
			require.Equal(t, td.id, id)
			require.Equal(t, td.value, value)
		})
	}
}

// //go:embed assets/testassets
// var testassets embed.FS

// type QueryModel struct {
// 	Query         string   `yaml:"query"`
// 	OperationName string   `yaml:"operationName"`
// 	Variables     string   `yaml:"variables"`
// 	Expect        []string `yaml:"expect"`
// }

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
// 				ID:        id,
// 				Source:    template,
// 				Operation: doc,
// 				Name:      meta.Name,
// 				Tags:      meta.Tags,
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
// 	for _, td := range readTestAssets(testassets, "assets/testassets", "test_") {
// 		t.Run(td.ID, func(t *testing.T) {
// 			rules := make(map[string]gqt.Doc, len(td.Templates))
// 			for _, r := range td.Templates {
// 				rules[r.ID] = r.Operation
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
