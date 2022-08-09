package engines_test

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"testing"

	"github.com/graph-guard/gguard-proxy/engines/rmap"
	"github.com/graph-guard/gguard-proxy/matcher"
	"github.com/graph-guard/gguard-proxy/utilities/xxhash"
	"github.com/graph-guard/gqt"
	"github.com/stretchr/testify/require"
)

func TestConstraintIdAndValue(t *testing.T) {
	for _, td := range []struct {
		input gqt.Constraint
		id    matcher.Constraint
		value any
		err   error
	}{
		{
			input: gqt.ConstraintMap{
				Constraint: new(gqt.Constraint),
			},
			id:    matcher.ConstraintMap,
			value: new(gqt.Constraint),
		},
		{
			input: gqt.ConstraintAny{},
			id:    matcher.ConstraintAny,
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
			id: matcher.ConstraintValEqual,
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
			id:    matcher.ConstraintValGreater,
			value: 42.0,
		},
		{
			input: gqt.ConstraintValLess{
				Value: 42.0,
			},
			id:    matcher.ConstraintValLess,
			value: 42.0,
		},
		{
			input: gqt.ConstraintValGreaterOrEqual{
				Value: 69.0,
			},
			id:    matcher.ConstraintValGreaterOrEqual,
			value: 69.0,
		},
		{
			input: gqt.ConstraintValLessOrEqual{
				Value: 69.0,
			},
			id:    matcher.ConstraintValLessOrEqual,
			value: 69.0,
		},
		{
			input: gqt.ConstraintBytelenEqual{
				Value: 1984,
			},
			id:    matcher.ConstraintBytelenEqual,
			value: uint(1984),
		},
		{
			input: gqt.ConstraintBytelenNotEqual{
				Value: 1984,
			},
			id:    matcher.ConstraintBytelenNotEqual,
			value: uint(1984),
		},
		{
			input: gqt.ConstraintBytelenGreater{
				Value: 282,
			},
			id:    matcher.ConstraintBytelenGreater,
			value: uint(282),
		},
		{
			input: gqt.ConstraintBytelenLess{
				Value: 282,
			},
			id:    matcher.ConstraintBytelenLess,
			value: uint(282),
		},
		{
			input: gqt.ConstraintBytelenGreaterOrEqual{
				Value: 27015,
			},
			id:    matcher.ConstraintBytelenGreaterOrEqual,
			value: uint(27015),
		},
		{
			input: gqt.ConstraintBytelenLessOrEqual{
				Value: 27015,
			},
			id:    matcher.ConstraintBytelenLessOrEqual,
			value: uint(27015),
		},
		{
			input: gqt.ConstraintLenEqual{
				Value: 997,
			},
			id:    matcher.ConstraintLenEqual,
			value: uint(997),
		},
		{
			input: gqt.ConstraintLenNotEqual{
				Value: 997,
			},
			id:    matcher.ConstraintLenNotEqual,
			value: uint(997),
		},
		{
			input: gqt.ConstraintLenGreater{
				Value: 47,
			},
			id:    matcher.ConstraintLenGreater,
			value: uint(47),
		},
		{
			input: gqt.ConstraintLenLess{
				Value: 47,
			},
			id:    matcher.ConstraintLenLess,
			value: uint(47),
		},
		{
			input: gqt.ConstraintLenGreaterOrEqual{
				Value: 404,
			},
			id:    matcher.ConstraintLenGreaterOrEqual,
			value: uint(404),
		},
		{
			input: gqt.ConstraintLenLessOrEqual{
				Value: 404,
			},
			id:    matcher.ConstraintLenLessOrEqual,
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

//go:embed assets/testassets/test_00/query.gql
var query_00 string

//go:embed assets/testassets/test_00/rule_00.txt
var rule_00_00 string

//go:embed assets/testassets/test_00/rule_01.txt
var rule_00_01 string

//go:embed assets/testassets/test_00/rule_02.txt
var rule_00_02 string

//go:embed assets/testassets/test_00/rule_03.txt
var rule_00_03 string

//go:embed assets/testassets/test_00/rule_04.txt
var rule_00_04 string

//go:embed assets/testassets/test_00/rule_05.txt
var rule_00_05 string

//go:embed assets/testassets/test_00b/query.gql
var query_00b string

//go:embed assets/testassets/test_00b/variables.json
var variables_00b string

//go:embed assets/testassets/test_00b/rule_00.txt
var rule_00b_00 string

//go:embed assets/testassets/test_00b/rule_01.txt
var rule_00b_01 string

//go:embed assets/testassets/test_00b/rule_02.txt
var rule_00b_02 string

//go:embed assets/testassets/test_00b/rule_03.txt
var rule_00b_03 string

//go:embed assets/testassets/test_00b/rule_04.txt
var rule_00b_04 string

//go:embed assets/testassets/test_00b/rule_05.txt
var rule_00b_05 string

//go:embed assets/testassets/test_01/query.gql
var query_01 string

//go:embed assets/testassets/test_01/rule_00.txt
var rule_01_00 string

//go:embed assets/testassets/test_02/query.gql
var query_02 string

//go:embed assets/testassets/test_02/rule_00.txt
var rule_02_00 string

//go:embed assets/testassets/test_03/query.gql
var query_03 string

//go:embed assets/testassets/test_03/rule_00.txt
var rule_03_00 string

//go:embed assets/testassets/test_04/query.gql
var query_04 string

//go:embed assets/testassets/test_04/rule_00.txt
var rule_04_00 string

//go:embed assets/testassets/test_05/query.gql
var query_05 string

//go:embed assets/testassets/test_05/rule_00.txt
var rule_05_00 string

//go:embed assets/testassets/test_05/rule_01.txt
var rule_05_01 string

//go:embed assets/testassets/test_06/query.gql
var query_06 string

//go:embed assets/testassets/test_06/rule_00.txt
var rule_06_00 string

//go:embed assets/testassets/test_06/rule_01.txt
var rule_06_01 string

//go:embed assets/testassets/test_07/query.gql
var query_07 string

//go:embed assets/testassets/test_07/rule_00.txt
var rule_07_00 string

//go:embed assets/testassets/test_08/query.gql
var query_08 string

//go:embed assets/testassets/test_08/rule_00.txt
var rule_08_00 string

//go:embed assets/testassets/test_09/query.gql
var query_09 string

//go:embed assets/testassets/test_09/rule_00.txt
var rule_09_00 string

//go:embed assets/testassets/test_10/query.gql
var query_10 string

//go:embed assets/testassets/test_10/rule_00.txt
var rule_10_00 string

//go:embed assets/testassets/test_11/query.gql
var query_11 string

//go:embed assets/testassets/test_11/rule_00.txt
var rule_11_00 string

//go:embed assets/testassets/test_11/rule_01.txt
var rule_11_01 string

//go:embed assets/testassets/test_11/rule_02.txt
var rule_11_02 string

//go:embed assets/testassets/test_12/query.gql
var query_12 string

//go:embed assets/testassets/test_12/rule_00.txt
var rule_12_00 string

//go:embed assets/testassets/test_13/query.gql
var query_13 string

//go:embed assets/testassets/test_13/rule_00.txt
var rule_13_00 string

//go:embed assets/testassets/test_14/query.gql
var query_14 string

//go:embed assets/testassets/test_14/rule_00.txt
var rule_14_00 string

//go:embed assets/testassets/test_15/query.gql
var query_15 string

//go:embed assets/testassets/test_15/rule_00.txt
var rule_15_00 string

func TestMatchAllRQmap(t *testing.T) {
	for _, td := range []struct {
		query         string
		operationName string
		variables     string
		rules         []string
		expect        []int
	}{
		{
			query:         query_00,
			operationName: "X",
			rules: []string{
				rule_00_00,
				rule_00_01,
				rule_00_02,
				rule_00_03,
				rule_00_04,
				rule_00_05,
			},
			expect: []int{0, 4},
		},
		{
			query:         query_00b,
			operationName: "X",
			variables:     variables_00b,
			rules: []string{
				rule_00b_00,
				rule_00b_01,
				rule_00b_02,
				rule_00b_03,
				rule_00b_04,
				rule_00b_05,
			},
			expect: []int{0, 4},
		},
		{
			query:         query_01,
			operationName: "X",
			rules: []string{
				rule_01_00,
			},
			expect: []int{},
		},
		{
			query:         query_02,
			operationName: "X",
			rules: []string{
				rule_02_00,
			},
			expect: []int{},
		},
		{
			query:         query_03,
			operationName: "X",
			rules: []string{
				rule_03_00,
			},
			expect: []int{},
		},
		{
			query:         query_04,
			operationName: "X",
			rules: []string{
				rule_04_00,
			},
			expect: []int{},
		},
		{
			query:         query_05,
			operationName: "X",
			rules: []string{
				rule_05_00,
				rule_05_01,
			},
			expect: []int{1},
		},
		{
			query:         query_06,
			operationName: "X",
			rules: []string{
				rule_06_00,
				rule_06_01,
			},
			expect: []int{1},
		},
		{
			query:         query_07,
			operationName: "X",
			rules: []string{
				rule_07_00,
			},
			expect: []int{0},
		},
		{
			query:         query_08,
			operationName: "X",
			rules: []string{
				rule_08_00,
			},
			expect: []int{0},
		},
		{
			query:         query_09,
			operationName: "X",
			rules: []string{
				rule_09_00,
			},
			expect: []int{0},
		},
		{
			query:         query_10,
			operationName: "X",
			rules: []string{
				rule_10_00,
			},
			expect: []int{0},
		},
		{
			query:         query_11,
			operationName: "X",
			rules: []string{
				rule_11_00,
				rule_11_01,
				rule_11_02,
			},
			expect: []int{2},
		},
		{
			query:         query_12,
			operationName: "X",
			rules: []string{
				rule_12_00,
			},
			expect: []int{},
		},
		{
			query:         query_13,
			operationName: "X",
			rules: []string{
				rule_13_00,
			},
			expect: []int{0},
		},
		{
			query:         query_14,
			operationName: "X",
			rules: []string{
				rule_14_00,
			},
			expect: []int{},
		},
		{
			query:         query_15,
			operationName: "X",
			rules: []string{
				rule_15_00,
			},
			expect: []int{0},
		},
	} {
		t.Run("", func(t *testing.T) {
			rules := make([]gqt.Doc, len(td.rules))
			for i, r := range td.rules {
				rd, err := gqt.Parse([]byte(r))
				require.False(t, err.IsErr())
				rules[i] = rd
			}

			rm, _ := rmap.New(rules, 0)

			actual := []int{}
			err := rm.MatchAll(
				context.Background(),
				[]byte(td.query),
				[]byte(td.operationName),
				[]byte(td.variables),
				func(n int) {
					actual = append(actual, n)
				},
			)
			require.NoError(t, err)
			require.Equal(t, td.expect, actual)
		})
	}
}

func TestPrintRQmap(t *testing.T) {
	for _, td := range []struct {
		rule   string
		expect string
	}{
		{
			rule: `
			query {
				a(
					a_0: val = {
						a_00: val = [val = 1, val = 2]
					}
				)
			}
			`,
			expect: fmt.Sprintf(`%d: 0
  variants:
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
			rule: `
			query {
				a(
					a_0: val = [ ... val = [ ... val <= 0 ] ]
				)
			}
			`,
			expect: fmt.Sprintf(`%d: 0
  variants:
    ConstraintMap: 0
      ConstraintMap:
        ConstraintValLessOrEqual:
          0
`, Hash("query.a.a_0")),
		},
		{
			rule: `
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
			expect: fmt.Sprintf(`%d: 0
  variants:
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

			rd, err := gqt.Parse([]byte(td.rule))
			require.False(t, err.IsErr())
			rm, _ := rmap.New([]gqt.Doc{rd}, 0)
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
