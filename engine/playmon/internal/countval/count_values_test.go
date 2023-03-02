package countval_test

import (
	"testing"

	"github.com/graph-guard/ggproxy/engine/playmon/internal/countval"
	"github.com/graph-guard/ggproxy/engine/playmon/internal/tokenreader"
	"github.com/graph-guard/ggproxy/gqlparse"
	"github.com/graph-guard/gqlscan"
	"github.com/stretchr/testify/require"
)

func TestCountValuesUntil(t *testing.T) {
	for _, tt := range []struct {
		Name         string
		Input        *tokenreader.Reader
		Term         gqlscan.Token
		ExpectValues int
		ExpectTokens int
	}{
		{
			Name: "sequence",
			Term: gqlscan.TokenArrEnd,
			Input: &tokenreader.Reader{
				Main: []gqlparse.Token{
					{ID: gqlscan.TokenStr, Value: []byte("text")},
					{ID: gqlscan.TokenEnumVal, Value: []byte("red")},
					{ID: gqlscan.TokenInt, Value: []byte("42")},
					{ID: gqlscan.TokenFloat, Value: []byte("3.1415")},
					{ID: gqlscan.TokenTrue},
					{ID: gqlscan.TokenFalse},
					{ID: gqlscan.TokenNull},
					{ID: gqlscan.TokenStrBlock},
					// {o:{f:[[null]]}}
					{ID: gqlscan.TokenObj},
					{ID: gqlscan.TokenObjField, Value: []byte("o")},
					{ID: gqlscan.TokenObj},
					{ID: gqlscan.TokenObjField, Value: []byte("f")},
					{ID: gqlscan.TokenArr},
					{ID: gqlscan.TokenArr},
					{ID: gqlscan.TokenNull},
					{ID: gqlscan.TokenArrEnd},
					{ID: gqlscan.TokenArrEnd},
					{ID: gqlscan.TokenObjEnd},
					{ID: gqlscan.TokenObjEnd},
					// {f:42}
					{ID: gqlscan.TokenObj},
					{ID: gqlscan.TokenObjField, Value: []byte("f")},
					{ID: gqlscan.TokenInt, Value: []byte("42")},
					{ID: gqlscan.TokenObjEnd},
					// []
					{ID: gqlscan.TokenArr},
					{ID: gqlscan.TokenArrEnd},
					// [[]]
					{ID: gqlscan.TokenArr},
					{ID: gqlscan.TokenArr},
					{ID: gqlscan.TokenArrEnd},
					{ID: gqlscan.TokenArrEnd},

					// Term
					{ID: gqlscan.TokenArrEnd},
				},
			},
			ExpectValues: 12,
			ExpectTokens: 29,
		},

		// {
		// 	Name: "string",
		// 	Input: []Token{
		// 		// "text"
		// 		{ID: gqlscan.TokenStr, Value: []byte("text")},
		// 	},
		// 	Expect: [][]Token{
		// 		{{ID: gqlscan.TokenStr, Value: []byte("text")}},
		// 	},
		// },
		// {
		// 	Name: "string block",
		// 	Input: []Token{
		// 		// """text"""
		// 		{ID: gqlscan.TokenStrBlock},
		// 	},
		// 	Expect: [][]Token{
		// 		{{ID: gqlscan.TokenStrBlock}},
		// 	},
		// },
		// {
		// 	Name: "float",
		// 	Input: []Token{
		// 		// 3.14
		// 		{ID: gqlscan.TokenFloat, Value: []byte("3.14")},
		// 	},
		// 	Expect: [][]Token{
		// 		{{ID: gqlscan.TokenFloat, Value: []byte("3.14")}},
		// 	},
		// },
		// {
		// 	Name: "enum",
		// 	Input: []Token{
		// 		// red
		// 		{ID: gqlscan.TokenEnumVal, Value: []byte("red")},
		// 	},
		// 	Expect: [][]Token{
		// 		{{ID: gqlscan.TokenEnumVal, Value: []byte("red")}},
		// 	},
		// },
		// {
		// 	Name: "boolean(true)",
		// 	Input: []Token{
		// 		// true
		// 		{ID: gqlscan.TokenTrue},
		// 	},
		// 	Expect: [][]Token{
		// 		{{ID: gqlscan.TokenTrue}},
		// 	},
		// },
		// {
		// 	Name: "boolean(false)",
		// 	Input: []Token{
		// 		// false
		// 		{ID: gqlscan.TokenFalse},
		// 	},
		// 	Expect: [][]Token{
		// 		{{ID: gqlscan.TokenFalse}},
		// 	},
		// },
		// {
		// 	Name: "null",
		// 	Input: []Token{
		// 		{ID: gqlscan.TokenNull},
		// 	},
		// 	Expect: [][]Token{
		// 		{{ID: gqlscan.TokenNull}},
		// 	},
		// },
		// {
		// 	Name: "object with array inside",
		// 	Input: []Token{
		// 		// {field:["text"]}
		// 		{ID: gqlscan.TokenObj},
		// 		{ID: gqlscan.TokenObjField, Value: []byte("field")},
		// 		{ID: gqlscan.TokenArr},
		// 		{ID: gqlscan.TokenStr, Value: []byte("text")},
		// 		{ID: gqlscan.TokenArrEnd},
		// 		{ID: gqlscan.TokenObjEnd},
		// 	},
		// 	Expect: [][]Token{
		// 		{
		// 			{ID: gqlscan.TokenObj},
		// 			{ID: gqlscan.TokenObjField, Value: []byte("field")},
		// 			{ID: gqlscan.TokenArr},
		// 			{ID: gqlscan.TokenStr, Value: []byte("text")},
		// 			{ID: gqlscan.TokenArrEnd},
		// 			{ID: gqlscan.TokenObjEnd},
		// 		},
		// 	},
		// },
		// {
		// 	Name: "object nested",
		// 	Input: []Token{
		// 		// {x:{y:{z:1}}}
		// 		{ID: gqlscan.TokenObj},
		// 		{ID: gqlscan.TokenObjField, Value: []byte("x")},
		// 		{ID: gqlscan.TokenObj},
		// 		{ID: gqlscan.TokenObjField, Value: []byte("y")},
		// 		{ID: gqlscan.TokenObj},
		// 		{ID: gqlscan.TokenObjField, Value: []byte("z")},
		// 		{ID: gqlscan.TokenInt, Value: []byte("1")},
		// 		{ID: gqlscan.TokenObjEnd},
		// 		{ID: gqlscan.TokenObjEnd},
		// 		{ID: gqlscan.TokenObjEnd},
		// 	},
		// 	Expect: [][]Token{
		// 		{
		// 			{ID: gqlscan.TokenObj},
		// 			{ID: gqlscan.TokenObjField, Value: []byte("x")},
		// 			{ID: gqlscan.TokenObj},
		// 			{ID: gqlscan.TokenObjField, Value: []byte("y")},
		// 			{ID: gqlscan.TokenObj},
		// 			{ID: gqlscan.TokenObjField, Value: []byte("z")},
		// 			{ID: gqlscan.TokenInt, Value: []byte("1")},
		// 			{ID: gqlscan.TokenObjEnd},
		// 			{ID: gqlscan.TokenObjEnd},
		// 			{ID: gqlscan.TokenObjEnd},
		// 		},
		// 	},
		// },
		// {
		// 	Name: "empty array",
		// 	Input: []Token{
		// 		// []
		// 		{ID: gqlscan.TokenArr},
		// 		{ID: gqlscan.TokenArrEnd},
		// 	},
		// 	Expect: [][]Token{},
		// },
		// {
		// 	Name: "array with 1 int",
		// 	Input: []Token{
		// 		// [42]
		// 		{ID: gqlscan.TokenArr},
		// 		{ID: gqlscan.TokenInt, Value: []byte("42")},
		// 		{ID: gqlscan.TokenArrEnd},
		// 	},
		// 	Expect: [][]Token{
		// 		{{ID: gqlscan.TokenInt, Value: []byte("42")}},
		// 	},
		// },
		// {
		// 	Name: "array with 3 int",
		// 	Input: []Token{
		// 		// [42, 0, 100500]
		// 		{ID: gqlscan.TokenArr},
		// 		{ID: gqlscan.TokenInt, Value: []byte("42")},
		// 		{ID: gqlscan.TokenInt, Value: []byte("0")},
		// 		{ID: gqlscan.TokenInt, Value: []byte("100500")},
		// 		{ID: gqlscan.TokenArrEnd},
		// 	},
		// 	Expect: [][]Token{
		// 		{{ID: gqlscan.TokenInt, Value: []byte("42")}},
		// 		{{ID: gqlscan.TokenInt, Value: []byte("0")}},
		// 		{{ID: gqlscan.TokenInt, Value: []byte("100500")}},
		// 	},
		// },
		// {
		// 	Name: "array with 1 string",
		// 	Input: []Token{
		// 		// ["text"]
		// 		{ID: gqlscan.TokenArr},
		// 		{ID: gqlscan.TokenStr, Value: []byte("text")},
		// 		{ID: gqlscan.TokenArrEnd},
		// 	},
		// 	Expect: [][]Token{
		// 		{{ID: gqlscan.TokenStr, Value: []byte("text")}},
		// 	},
		// },
		// {
		// 	Name: "array with 1 string block",
		// 	Input: []Token{
		// 		// ["""text"""]
		// 		{ID: gqlscan.TokenArr},
		// 		{ID: gqlscan.TokenStrBlock},
		// 		{ID: gqlscan.TokenArrEnd},
		// 	},
		// 	Expect: [][]Token{
		// 		{{ID: gqlscan.TokenStrBlock}},
		// 	},
		// },
		// {
		// 	Name: "array with 1 float",
		// 	Input: []Token{
		// 		// [3.14]
		// 		{ID: gqlscan.TokenArr},
		// 		{ID: gqlscan.TokenFloat, Value: []byte("3.14")},
		// 		{ID: gqlscan.TokenArrEnd},
		// 	},
		// 	Expect: [][]Token{
		// 		{{ID: gqlscan.TokenFloat, Value: []byte("3.14")}},
		// 	},
		// },
		// {
		// 	Name: "array with 1 enum",
		// 	Input: []Token{
		// 		// [red]
		// 		{ID: gqlscan.TokenArr},
		// 		{ID: gqlscan.TokenEnumVal, Value: []byte("red")},
		// 		{ID: gqlscan.TokenArrEnd},
		// 	},
		// 	Expect: [][]Token{
		// 		{{ID: gqlscan.TokenEnumVal, Value: []byte("red")}},
		// 	},
		// },
		// {
		// 	Name: "array with 1 boolean(true)",
		// 	Input: []Token{
		// 		// [true]
		// 		{ID: gqlscan.TokenArr},
		// 		{ID: gqlscan.TokenTrue},
		// 		{ID: gqlscan.TokenArrEnd},
		// 	},
		// 	Expect: [][]Token{
		// 		{{ID: gqlscan.TokenTrue}},
		// 	},
		// },
		// {
		// 	Name: "array with 1 boolean(false)",
		// 	Input: []Token{
		// 		// [false]
		// 		{ID: gqlscan.TokenArr},
		// 		{ID: gqlscan.TokenFalse},
		// 		{ID: gqlscan.TokenArrEnd},
		// 	},
		// 	Expect: [][]Token{
		// 		{{ID: gqlscan.TokenFalse}},
		// 	},
		// },
		// {
		// 	Name: "array with 1 null",
		// 	Input: []Token{
		// 		// [null]
		// 		{ID: gqlscan.TokenArr},
		// 		{ID: gqlscan.TokenNull},
		// 		{ID: gqlscan.TokenArrEnd},
		// 	},
		// 	Expect: [][]Token{
		// 		{{ID: gqlscan.TokenNull}},
		// 	},
		// },
		// {
		// 	Name: "array with 1 object",
		// 	Input: []Token{
		// 		// [{x:0}]
		// 		{ID: gqlscan.TokenArr},
		// 		{ID: gqlscan.TokenObj},
		// 		{ID: gqlscan.TokenObjField, Value: []byte("x")},
		// 		{ID: gqlscan.TokenInt, Value: []byte("0")},
		// 		{ID: gqlscan.TokenObjEnd},
		// 		{ID: gqlscan.TokenArrEnd},
		// 	},
		// 	Expect: [][]Token{
		// 		{
		// 			{ID: gqlscan.TokenObj},
		// 			{ID: gqlscan.TokenObjField, Value: []byte("x")},
		// 			{ID: gqlscan.TokenInt, Value: []byte("0")},
		// 			{ID: gqlscan.TokenObjEnd},
		// 		},
		// 	},
		// },
		// {
		// 	Name: "3d array string",
		// 	Input: []Token{
		// 		// [[["1"],["2"]],[["3"]]]
		// 		{ID: gqlscan.TokenArr},
		// 		{ID: gqlscan.TokenArr},
		// 		{ID: gqlscan.TokenArr},
		// 		{ID: gqlscan.TokenStr, Value: []byte("1")},
		// 		{ID: gqlscan.TokenArrEnd},
		// 		{ID: gqlscan.TokenArr},
		// 		{ID: gqlscan.TokenStr, Value: []byte("2")},
		// 		{ID: gqlscan.TokenArrEnd},
		// 		{ID: gqlscan.TokenArrEnd},
		// 		{ID: gqlscan.TokenArr},
		// 		{ID: gqlscan.TokenArr},
		// 		{ID: gqlscan.TokenStr, Value: []byte("3")},
		// 		{ID: gqlscan.TokenArrEnd},
		// 		{ID: gqlscan.TokenArrEnd},
		// 		{ID: gqlscan.TokenArrEnd},
		// 	},
		// 	Expect: [][]Token{
		// 		{{ID: gqlscan.TokenStr, Value: []byte("1")}},
		// 		{{ID: gqlscan.TokenStr, Value: []byte("2")}},
		// 		{{ID: gqlscan.TokenStr, Value: []byte("3")}},
		// 	},
		// },
		// {
		// 	Name: "array with null and nested object with array inside",
		// 	Input: []Token{
		// 		// [null, {object:{array:["text"]}}]
		// 		{ID: gqlscan.TokenArr},
		// 		{ID: gqlscan.TokenNull},
		// 		{ID: gqlscan.TokenObj},
		// 		{ID: gqlscan.TokenObjField, Value: []byte("object")},
		// 		{ID: gqlscan.TokenObj},
		// 		{ID: gqlscan.TokenObjField, Value: []byte("array")},
		// 		{ID: gqlscan.TokenArr},
		// 		{ID: gqlscan.TokenStr, Value: []byte("text")},
		// 		{ID: gqlscan.TokenArrEnd},
		// 		{ID: gqlscan.TokenObjEnd},
		// 		{ID: gqlscan.TokenObjEnd},
		// 		{ID: gqlscan.TokenArrEnd},
		// 	},
		// 	Expect: [][]Token{
		// 		{{ID: gqlscan.TokenNull}},
		// 		{
		// 			{ID: gqlscan.TokenObj},
		// 			{ID: gqlscan.TokenObjField, Value: []byte("object")},
		// 			{ID: gqlscan.TokenObj},
		// 			{ID: gqlscan.TokenObjField, Value: []byte("array")},
		// 			{ID: gqlscan.TokenArr},
		// 			{ID: gqlscan.TokenStr, Value: []byte("text")},
		// 			{ID: gqlscan.TokenArrEnd},
		// 			{ID: gqlscan.TokenObjEnd},
		// 			{ID: gqlscan.TokenObjEnd},
		// 		},
		// 	},
		// },
	} {
		t.Run(tt.Name, func(t *testing.T) {
			values, tokens := countval.Until(tt.Input, tt.Term)
			require.Equal(t, tt.ExpectValues, values)
			require.Equal(t, tt.ExpectTokens, tokens)
		})
	}
}
