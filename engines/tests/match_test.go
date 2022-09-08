package engines_test

import (
	"bytes"
	"embed"
	_ "embed"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"

	"github.com/graph-guard/ggproxy/config"
	"github.com/graph-guard/ggproxy/config/metadata"
	"github.com/graph-guard/ggproxy/engines/rmap"
	"github.com/graph-guard/ggproxy/gqlparse"
	"github.com/graph-guard/ggproxy/utilities/xxhash"
	"github.com/graph-guard/gqt"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestConstraintIdAndValue(t *testing.T) {
	for _, td := range []struct {
		input gqt.Constraint
		id    rmap.Constraint
		value any
		err   error
	}{
		{
			input: gqt.ConstraintMap{
				Constraint: new(gqt.Constraint),
			},
			id:    rmap.ConstraintMap,
			value: new(gqt.Constraint),
		},
		{
			input: gqt.ConstraintAny{},
			id:    rmap.ConstraintAny,
			value: nil,
		},
		{
			input: gqt.ConstraintValEqual{
				Value: gqt.ValueObject{
					Fields: []gqt.ObjectField{
						{
							Name: "a",
							Value: gqt.ConstraintValLessOrEqual{
								Value: 42.0,
							},
						},
					},
				},
			},
			id: rmap.ConstraintValEqual,
			value: gqt.ValueObject{
				Fields: []gqt.ObjectField{
					{
						Name: "a",
						Value: gqt.ConstraintValLessOrEqual{
							Value: 42.0,
						},
					},
				},
			},
		},
		{
			input: gqt.ConstraintValGreater{
				Value: 42.0,
			},
			id:    rmap.ConstraintValGreater,
			value: 42.0,
		},
		{
			input: gqt.ConstraintValLess{
				Value: 42.0,
			},
			id:    rmap.ConstraintValLess,
			value: 42.0,
		},
		{
			input: gqt.ConstraintValGreaterOrEqual{
				Value: 69.0,
			},
			id:    rmap.ConstraintValGreaterOrEqual,
			value: 69.0,
		},
		{
			input: gqt.ConstraintValLessOrEqual{
				Value: 69.0,
			},
			id:    rmap.ConstraintValLessOrEqual,
			value: 69.0,
		},
		{
			input: gqt.ConstraintBytelenEqual{
				Value: 1984,
			},
			id:    rmap.ConstraintBytelenEqual,
			value: uint(1984),
		},
		{
			input: gqt.ConstraintBytelenNotEqual{
				Value: 1984,
			},
			id:    rmap.ConstraintBytelenNotEqual,
			value: uint(1984),
		},
		{
			input: gqt.ConstraintBytelenGreater{
				Value: 282,
			},
			id:    rmap.ConstraintBytelenGreater,
			value: uint(282),
		},
		{
			input: gqt.ConstraintBytelenLess{
				Value: 282,
			},
			id:    rmap.ConstraintBytelenLess,
			value: uint(282),
		},
		{
			input: gqt.ConstraintBytelenGreaterOrEqual{
				Value: 27015,
			},
			id:    rmap.ConstraintBytelenGreaterOrEqual,
			value: uint(27015),
		},
		{
			input: gqt.ConstraintBytelenLessOrEqual{
				Value: 27015,
			},
			id:    rmap.ConstraintBytelenLessOrEqual,
			value: uint(27015),
		},
		{
			input: gqt.ConstraintLenEqual{
				Value: 997,
			},
			id:    rmap.ConstraintLenEqual,
			value: uint(997),
		},
		{
			input: gqt.ConstraintLenNotEqual{
				Value: 997,
			},
			id:    rmap.ConstraintLenNotEqual,
			value: uint(997),
		},
		{
			input: gqt.ConstraintLenGreater{
				Value: 47,
			},
			id:    rmap.ConstraintLenGreater,
			value: uint(47),
		},
		{
			input: gqt.ConstraintLenLess{
				Value: 47,
			},
			id:    rmap.ConstraintLenLess,
			value: uint(47),
		},
		{
			input: gqt.ConstraintLenGreaterOrEqual{
				Value: 404,
			},
			id:    rmap.ConstraintLenGreaterOrEqual,
			value: uint(404),
		},
		{
			input: gqt.ConstraintLenLessOrEqual{
				Value: 404,
			},
			id:    rmap.ConstraintLenLessOrEqual,
			value: uint(404),
		},
	} {
		t.Run("", func(t *testing.T) {
			id, value := rmap.ConstraintIdAndValue(td.input)
			require.Equal(t, td.id, id)
			require.Equal(t, td.value, value)
		})
	}
}

//go:embed assets/testassets
var testassets embed.FS

type QueryModel struct {
	Query         string   `yaml:"query"`
	OperationName string   `yaml:"operationName"`
	Variables     string   `yaml:"variables"`
	Expect        []string `yaml:"expect"`
}

type MatchTest struct {
	ID string
	*QueryModel
	Templates []*config.Template
}

func readTestAsset(
	filesystem fs.FS, path string,
) (
	query *QueryModel, templates []*config.Template,
) {
	test, err := fs.ReadDir(filesystem, path)
	if err != nil {
		panic(err)
	}

	for _, f := range test {
		if f.IsDir() {
			continue
		}
		fn := f.Name()
		fp := filepath.Join(path, f.Name())
		if strings.HasSuffix(fn, ".gqt") {
			id := strings.ToLower(fn[:len(fn)-len(filepath.Ext(fn))])
			src, err := filesystem.Open(fp)
			if err != nil {
				panic(err)
			}
			b, err := io.ReadAll(src)
			if err != nil {
				panic(err)
			}

			meta, template, err := metadata.Parse(b)
			if err != nil {
				panic(err)
			}
			doc, errParser := gqt.Parse(template)
			if errParser.IsErr() {
				panic(errParser)
			}

			templates = append(templates, &config.Template{
				ID:       id,
				Source:   template,
				Document: doc,
				Name:     meta.Name,
				Tags:     meta.Tags,
			})
		}
		if strings.HasSuffix(fn, ".yml") || strings.HasSuffix(fn, ".yaml") {
			src, err := filesystem.Open(fp)
			if err != nil {
				panic(err)
			}
			d := yaml.NewDecoder(src)
			d.KnownFields(true)
			err = d.Decode(&query)
			if err != nil {
				panic(err)
			}
		}
	}

	return
}

func readTestAssets(filesystem fs.FS, path, prefix string) (assets []*MatchTest) {
	root, err := fs.ReadDir(filesystem, path)
	if err != nil {
		panic(err)
	}
	for _, testDir := range root {
		if !testDir.IsDir() {
			continue
		}
		testDirName := testDir.Name()
		testDirPath := filepath.Join(path, testDirName)
		if !strings.HasPrefix(testDirName, prefix) {
			continue
		}

		query, templates := readTestAsset(filesystem, testDirPath)
		assets = append(assets, &MatchTest{
			ID:         testDirName,
			QueryModel: query,
			Templates:  templates,
		})
	}

	return
}

func TestMatchAllPartedQuery(t *testing.T) {
	for _, td := range readTestAssets(testassets, "assets/testassets", "test_") {
		t.Run(td.ID, func(t *testing.T) {
			rules := make(map[string]gqt.Doc, len(td.Templates))
			for _, r := range td.Templates {
				rules[r.ID] = r.Document
			}

			p := gqlparse.NewParser()
			rm, _ := rmap.New(rules, 0)

			p.Parse(
				[]byte(td.Query),
				[]byte(td.OperationName),
				[]byte(td.Variables),
				func(
					varVals [][]gqlparse.Token,
					operation []gqlparse.Token,
					selectionSet []gqlparse.Token,
				) {
					actual := []string{}
					rm.MatchAll(
						varVals,
						operation[0].ID,
						selectionSet,
						func(id string) {
							actual = append(actual, id)
						},
					)
					require.Len(t, actual, len(td.Expect))
					for _, e := range td.Expect {
						require.Contains(t, actual, e)
					}
				},
				func(err error) {
					t.Fatalf("unexpected error: %v", err)
				},
			)
		})
	}
}

func TestPrintPartedQuery(t *testing.T) {
	for _, td := range []struct {
		template string
		expect   string
	}{
		{
			template: `
			query {
				a(
					a_0: val = 0
					a_1: val = "a"
				)
			}
			`,
			expect: fmt.Sprintf(`%d:
    ConstraintValEqual: 0
      0
%d:
    ConstraintValEqual: 0
      a
`, Hash("query.a.a_0"), Hash("query.a.a_1")),
		},
		{
			template: `
			query {
				a(
					a_0: val = {
						a_00: val = [val = 1, val = 2]
					}
				)
			}
			`,
			expect: fmt.Sprintf(`%d:
    ConstraintValEqual: 0
      -:
        ConstraintValEqual:
          1
      -:
        ConstraintValEqual:
          2
`, Hash("query.a.a_0.a_00")),
		},
		{
			template: `
			query {
				a(
					a_0: val = [ ... val = [ ... val <= 0 ] ]
				)
			}
			`,
			expect: fmt.Sprintf(`%d:
    ConstraintMap: 0
      ConstraintMap:
        ConstraintValLessOrEqual:
          0
`, Hash("query.a.a_0")),
		},
		{
			template: `
			query {
				a(
					a_0: val = [
						val = {
							a_000: val > 5
						}
						val = {
							a_010: val = [val = 0, val = 1]
						}
					]
				)
			}
			`,
			expect: fmt.Sprintf(`%d:
    ConstraintValEqual: 0
      -:
        ConstraintValEqual:
          a_000:
            ConstraintValGreater:
              5
      -:
        ConstraintValEqual:
          a_010:
            ConstraintValEqual:
              -:
                ConstraintValEqual:
                  0
              -:
                ConstraintValEqual:
                  1
`, Hash("query.a.a_0")),
		},
	} {
		t.Run("", func(t *testing.T) {
			b := new(bytes.Buffer)

			rd, err := gqt.Parse([]byte(td.template))
			require.False(t, err.IsErr())
			rm, _ := rmap.New(map[string]gqt.Doc{
				"rd": rd,
			}, 0)
			rm.Print(b)

			require.Equal(t, td.expect, b.String())
		})
	}
}

func Hash(s string) uint64 {
	h := xxhash.New(0)
	xxhash.Write(&h, s)
	return h.Sum64()
}
