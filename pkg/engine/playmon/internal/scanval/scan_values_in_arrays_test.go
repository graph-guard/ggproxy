package scanval_test

import (
	"fmt"
	"testing"

	"github.com/graph-guard/ggproxy/pkg/engine/playmon/internal/scanval"
	"github.com/graph-guard/ggproxy/pkg/engine/playmon/internal/tokenreader"
	"github.com/graph-guard/ggproxy/pkg/gqlparse"
	"github.com/graph-guard/gqlscan"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInValues(t *testing.T) {
	for _, tt := range []struct {
		Name   string
		Input  []gqlparse.Token
		Expect [][]gqlparse.Token
	}{
		{
			Name: "string",
			Input: []gqlparse.Token{
				// "text"
				{ID: gqlscan.TokenStr, Value: []byte("text")},
			},
			Expect: [][]gqlparse.Token{
				{{ID: gqlscan.TokenStr, Value: []byte("text")}},
			},
		},
		{
			Name: "string_block",
			Input: []gqlparse.Token{
				// """text"""
				{ID: gqlscan.TokenStrBlock},
			},
			Expect: [][]gqlparse.Token{
				{{ID: gqlscan.TokenStrBlock}},
			},
		},
		{
			Name: "float",
			Input: []gqlparse.Token{
				// 3.14
				{ID: gqlscan.TokenFloat, Value: []byte("3.14")},
			},
			Expect: [][]gqlparse.Token{
				{{ID: gqlscan.TokenFloat, Value: []byte("3.14")}},
			},
		},
		{
			Name: "enum",
			Input: []gqlparse.Token{
				// red
				{ID: gqlscan.TokenEnumVal, Value: []byte("red")},
			},
			Expect: [][]gqlparse.Token{
				{{ID: gqlscan.TokenEnumVal, Value: []byte("red")}},
			},
		},
		{
			Name: "boolean(true)",
			Input: []gqlparse.Token{
				// true
				{ID: gqlscan.TokenTrue},
			},
			Expect: [][]gqlparse.Token{
				{{ID: gqlscan.TokenTrue}},
			},
		},
		{
			Name: "boolean(false)",
			Input: []gqlparse.Token{
				// false
				{ID: gqlscan.TokenFalse},
			},
			Expect: [][]gqlparse.Token{
				{{ID: gqlscan.TokenFalse}},
			},
		},
		{
			Name: "null",
			Input: []gqlparse.Token{
				{ID: gqlscan.TokenNull},
			},
			Expect: [][]gqlparse.Token{
				{{ID: gqlscan.TokenNull}},
			},
		},
		{
			Name: "object_with_array_inside",
			Input: []gqlparse.Token{
				// {field:["text"]}
				{ID: gqlscan.TokenObj},
				{ID: gqlscan.TokenObjField, Value: []byte("field")},
				{ID: gqlscan.TokenArr},
				{ID: gqlscan.TokenStr, Value: []byte("text")},
				{ID: gqlscan.TokenArrEnd},
				{ID: gqlscan.TokenObjEnd},
			},
			Expect: [][]gqlparse.Token{
				{
					{ID: gqlscan.TokenObj},
					{ID: gqlscan.TokenObjField, Value: []byte("field")},
					{ID: gqlscan.TokenArr},
					{ID: gqlscan.TokenStr, Value: []byte("text")},
					{ID: gqlscan.TokenArrEnd},
					{ID: gqlscan.TokenObjEnd},
				},
			},
		},
		{
			Name: "object_nested",
			Input: []gqlparse.Token{
				// {x:{y:{z:1}}}
				{ID: gqlscan.TokenObj},
				{ID: gqlscan.TokenObjField, Value: []byte("x")},
				{ID: gqlscan.TokenObj},
				{ID: gqlscan.TokenObjField, Value: []byte("y")},
				{ID: gqlscan.TokenObj},
				{ID: gqlscan.TokenObjField, Value: []byte("z")},
				{ID: gqlscan.TokenInt, Value: []byte("1")},
				{ID: gqlscan.TokenObjEnd},
				{ID: gqlscan.TokenObjEnd},
				{ID: gqlscan.TokenObjEnd},
			},
			Expect: [][]gqlparse.Token{
				{
					{ID: gqlscan.TokenObj},
					{ID: gqlscan.TokenObjField, Value: []byte("x")},
					{ID: gqlscan.TokenObj},
					{ID: gqlscan.TokenObjField, Value: []byte("y")},
					{ID: gqlscan.TokenObj},
					{ID: gqlscan.TokenObjField, Value: []byte("z")},
					{ID: gqlscan.TokenInt, Value: []byte("1")},
					{ID: gqlscan.TokenObjEnd},
					{ID: gqlscan.TokenObjEnd},
					{ID: gqlscan.TokenObjEnd},
				},
			},
		},
		{
			Name: "empty_array",
			Input: []gqlparse.Token{
				// []
				{ID: gqlscan.TokenArr},
				{ID: gqlscan.TokenArrEnd},
			},
			Expect: [][]gqlparse.Token{},
		},
		{
			Name: "array_with_1_int",
			Input: []gqlparse.Token{
				// [42]
				{ID: gqlscan.TokenArr},
				{ID: gqlscan.TokenInt, Value: []byte("42")},
				{ID: gqlscan.TokenArrEnd},
			},
			Expect: [][]gqlparse.Token{
				{{ID: gqlscan.TokenInt, Value: []byte("42")}},
			},
		},
		{
			Name: "array_with_3_int",
			Input: []gqlparse.Token{
				// [42, 0, 100500]
				{ID: gqlscan.TokenArr},
				{ID: gqlscan.TokenInt, Value: []byte("42")},
				{ID: gqlscan.TokenInt, Value: []byte("0")},
				{ID: gqlscan.TokenInt, Value: []byte("100500")},
				{ID: gqlscan.TokenArrEnd},
			},
			Expect: [][]gqlparse.Token{
				{{ID: gqlscan.TokenInt, Value: []byte("42")}},
				{{ID: gqlscan.TokenInt, Value: []byte("0")}},
				{{ID: gqlscan.TokenInt, Value: []byte("100500")}},
			},
		},
		{
			Name: "array_with_1_string",
			Input: []gqlparse.Token{
				// ["text"]
				{ID: gqlscan.TokenArr},
				{ID: gqlscan.TokenStr, Value: []byte("text")},
				{ID: gqlscan.TokenArrEnd},
			},
			Expect: [][]gqlparse.Token{
				{{ID: gqlscan.TokenStr, Value: []byte("text")}},
			},
		},
		{
			Name: "array_with_1_string_block",
			Input: []gqlparse.Token{
				// ["""text"""]
				{ID: gqlscan.TokenArr},
				{ID: gqlscan.TokenStrBlock},
				{ID: gqlscan.TokenArrEnd},
			},
			Expect: [][]gqlparse.Token{
				{{ID: gqlscan.TokenStrBlock}},
			},
		},
		{
			Name: "array_with_1_float",
			Input: []gqlparse.Token{
				// [3.14]
				{ID: gqlscan.TokenArr},
				{ID: gqlscan.TokenFloat, Value: []byte("3.14")},
				{ID: gqlscan.TokenArrEnd},
			},
			Expect: [][]gqlparse.Token{
				{{ID: gqlscan.TokenFloat, Value: []byte("3.14")}},
			},
		},
		{
			Name: "array_with_1_enum",
			Input: []gqlparse.Token{
				// [red]
				{ID: gqlscan.TokenArr},
				{ID: gqlscan.TokenEnumVal, Value: []byte("red")},
				{ID: gqlscan.TokenArrEnd},
			},
			Expect: [][]gqlparse.Token{
				{{ID: gqlscan.TokenEnumVal, Value: []byte("red")}},
			},
		},
		{
			Name: "array_with_1_boolean(true)",
			Input: []gqlparse.Token{
				// [true]
				{ID: gqlscan.TokenArr},
				{ID: gqlscan.TokenTrue},
				{ID: gqlscan.TokenArrEnd},
			},
			Expect: [][]gqlparse.Token{
				{{ID: gqlscan.TokenTrue}},
			},
		},
		{
			Name: "array_with_1_boolean(false)",
			Input: []gqlparse.Token{
				// [false]
				{ID: gqlscan.TokenArr},
				{ID: gqlscan.TokenFalse},
				{ID: gqlscan.TokenArrEnd},
			},
			Expect: [][]gqlparse.Token{
				{{ID: gqlscan.TokenFalse}},
			},
		},
		{
			Name: "array_with_1_null",
			Input: []gqlparse.Token{
				// [null]
				{ID: gqlscan.TokenArr},
				{ID: gqlscan.TokenNull},
				{ID: gqlscan.TokenArrEnd},
			},
			Expect: [][]gqlparse.Token{
				{{ID: gqlscan.TokenNull}},
			},
		},
		{
			Name: "array_with_1_object",
			Input: []gqlparse.Token{
				// [{x:0}]
				{ID: gqlscan.TokenArr},
				{ID: gqlscan.TokenObj},
				{ID: gqlscan.TokenObjField, Value: []byte("x")},
				{ID: gqlscan.TokenInt, Value: []byte("0")},
				{ID: gqlscan.TokenObjEnd},
				{ID: gqlscan.TokenArrEnd},
			},
			Expect: [][]gqlparse.Token{
				{
					{ID: gqlscan.TokenObj},
					{ID: gqlscan.TokenObjField, Value: []byte("x")},
					{ID: gqlscan.TokenInt, Value: []byte("0")},
					{ID: gqlscan.TokenObjEnd},
				},
			},
		},
		{
			Name: "3d_array_string",
			Input: []gqlparse.Token{
				// [[["1"],["2"]],[["3"]]]
				{ID: gqlscan.TokenArr},
				{ID: gqlscan.TokenArr},
				{ID: gqlscan.TokenArr},
				{ID: gqlscan.TokenStr, Value: []byte("1")},
				{ID: gqlscan.TokenArrEnd},
				{ID: gqlscan.TokenArr},
				{ID: gqlscan.TokenStr, Value: []byte("2")},
				{ID: gqlscan.TokenArrEnd},
				{ID: gqlscan.TokenArrEnd},
				{ID: gqlscan.TokenArr},
				{ID: gqlscan.TokenArr},
				{ID: gqlscan.TokenStr, Value: []byte("3")},
				{ID: gqlscan.TokenArrEnd},
				{ID: gqlscan.TokenArrEnd},
				{ID: gqlscan.TokenArrEnd},
			},
			Expect: [][]gqlparse.Token{
				{{ID: gqlscan.TokenStr, Value: []byte("1")}},
				{{ID: gqlscan.TokenStr, Value: []byte("2")}},
				{{ID: gqlscan.TokenStr, Value: []byte("3")}},
			},
		},
		{
			Name: "array_with_null_and_nested_object_with_array_inside",
			Input: []gqlparse.Token{
				// [null, {object:{array:["text"]}}]
				{ID: gqlscan.TokenArr},
				{ID: gqlscan.TokenNull},
				{ID: gqlscan.TokenObj},
				{ID: gqlscan.TokenObjField, Value: []byte("object")},
				{ID: gqlscan.TokenObj},
				{ID: gqlscan.TokenObjField, Value: []byte("array")},
				{ID: gqlscan.TokenArr},
				{ID: gqlscan.TokenStr, Value: []byte("text")},
				{ID: gqlscan.TokenArrEnd},
				{ID: gqlscan.TokenObjEnd},
				{ID: gqlscan.TokenObjEnd},
				{ID: gqlscan.TokenArrEnd},
			},
			Expect: [][]gqlparse.Token{
				{{ID: gqlscan.TokenNull}},
				{
					{ID: gqlscan.TokenObj},
					{ID: gqlscan.TokenObjField, Value: []byte("object")},
					{ID: gqlscan.TokenObj},
					{ID: gqlscan.TokenObjField, Value: []byte("array")},
					{ID: gqlscan.TokenArr},
					{ID: gqlscan.TokenStr, Value: []byte("text")},
					{ID: gqlscan.TokenArrEnd},
					{ID: gqlscan.TokenObjEnd},
					{ID: gqlscan.TokenObjEnd},
				},
			},
		},
	} {
		t.Run(tt.Name, func(t *testing.T) {
			actual := [][]gqlparse.Token{}
			actualCounter := 0
			stopped := scanval.InArrays(
				&tokenreader.Reader{Main: tt.Input},
				func(r *tokenreader.Reader) (stop bool) {
					var cp []gqlparse.Token
					for i := 0; !r.EOF() && i < len(tt.Expect[actualCounter]); i++ {
						cp = append(cp, r.ReadOne())
					}
					actual = append(actual, cp)
					actualCounter++
					return false
				},
			)
			// Manually print diff for better readability
			printSet := func(title string, t [][]gqlparse.Token) {
				fmt.Printf("%s (%d)\n", title, len(t))
				for i, t := range t {
					fmt.Printf(" item %d (%d token(s)):\n", i, len(t))
					for i, t := range t {
						fmt.Printf("  %d: %s\n", i, t.String())
					}
				}
			}
			isEqual := assert.ObjectsAreEqual(tt.Expect, actual)
			if !isEqual {
				printSet("expect", tt.Expect)
				printSet("actual", actual)
			}
			require.True(t, isEqual)
			require.False(t, stopped)
			require.Equal(t, tt.Expect, actual)
		})
	}
}
