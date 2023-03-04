package scanval_test

import (
	"testing"

	"github.com/graph-guard/ggproxy/pkg/engine/playmon/internal/scanval"
	"github.com/graph-guard/ggproxy/pkg/engine/playmon/internal/tokenreader"
	"github.com/graph-guard/ggproxy/pkg/gqlparse"
	"github.com/graph-guard/gqlscan"
	"github.com/stretchr/testify/require"
)

func TestLength(t *testing.T) {
	for _, tt := range []struct {
		Name   string
		Input  []gqlparse.Token
		Expect int
	}{
		{
			Name:   "bool_true",
			Input:  []gqlparse.Token{{ID: gqlscan.TokenTrue}},
			Expect: 1,
		},
		{
			Name:   "bool_false",
			Input:  []gqlparse.Token{{ID: gqlscan.TokenFalse}},
			Expect: 1,
		},
		{
			Name:   "int",
			Input:  []gqlparse.Token{{ID: gqlscan.TokenInt, Value: []byte("-123")}},
			Expect: 1,
		},
		{
			Name:   "float",
			Input:  []gqlparse.Token{{ID: gqlscan.TokenFloat, Value: []byte("-3.14")}},
			Expect: 1,
		},
		{
			Name:   "string",
			Input:  []gqlparse.Token{{ID: gqlscan.TokenStr, Value: []byte("text")}},
			Expect: 1,
		},
		{
			Name:   "string_block",
			Input:  []gqlparse.Token{{ID: gqlscan.TokenStrBlock}},
			Expect: 1,
		},
		{
			Name:   "enum",
			Input:  []gqlparse.Token{{ID: gqlscan.TokenEnumVal, Value: []byte("red")}},
			Expect: 1,
		},
		{
			Name:   "null",
			Input:  []gqlparse.Token{{ID: gqlscan.TokenNull}},
			Expect: 1,
		},
		{
			Name: "array",
			Input: []gqlparse.Token{
				// []
				{ID: gqlscan.TokenArr},
				{ID: gqlscan.TokenArrEnd},
			},
			Expect: 2,
		},
		{
			Name: "array_int_null",
			Input: []gqlparse.Token{
				// [1,null,3]
				{ID: gqlscan.TokenArr},
				{ID: gqlscan.TokenInt, Value: []byte("1")},
				{ID: gqlscan.TokenNull},
				{ID: gqlscan.TokenInt, Value: []byte("3")},
				{ID: gqlscan.TokenArrEnd},
			},
			Expect: 5,
		},
		{
			Name: "array_2d_int_null",
			Input: []gqlparse.Token{
				// [[1],[],[null],null]
				{ID: gqlscan.TokenArr},
				{ID: gqlscan.TokenArr},
				{ID: gqlscan.TokenInt, Value: []byte("1")},
				{ID: gqlscan.TokenArrEnd},

				{ID: gqlscan.TokenArr},
				{ID: gqlscan.TokenArrEnd},

				{ID: gqlscan.TokenArr},
				{ID: gqlscan.TokenNull},
				{ID: gqlscan.TokenArrEnd},
				{ID: gqlscan.TokenNull},
				{ID: gqlscan.TokenArrEnd},
			},
			Expect: 11,
		},
		{
			Name: "object",
			Input: []gqlparse.Token{
				// {foo:42}
				{ID: gqlscan.TokenObj},
				{ID: gqlscan.TokenObjField, Value: []byte("foo")},
				{ID: gqlscan.TokenInt, Value: []byte("42")},
				{ID: gqlscan.TokenObjEnd},
			},
			Expect: 4,
		},
		{
			Name: "object_nested",
			Input: []gqlparse.Token{
				// {foo:{bar:42}}
				{ID: gqlscan.TokenObj},
				{ID: gqlscan.TokenObjField, Value: []byte("foo")},
				{ID: gqlscan.TokenObj},
				{ID: gqlscan.TokenObjField, Value: []byte("bar")},
				{ID: gqlscan.TokenInt, Value: []byte("42")},
				{ID: gqlscan.TokenObjEnd},
				{ID: gqlscan.TokenObjEnd},
			},
			Expect: 7,
		},
		{
			Name: "object_nested_with_array",
			Input: []gqlparse.Token{
				// {foo:{bar:[42]}}
				{ID: gqlscan.TokenObj},
				{ID: gqlscan.TokenObjField, Value: []byte("foo")},
				{ID: gqlscan.TokenObj},
				{ID: gqlscan.TokenObjField, Value: []byte("bar")},
				{ID: gqlscan.TokenArr},
				{ID: gqlscan.TokenInt, Value: []byte("42")},
				{ID: gqlscan.TokenArrEnd},
				{ID: gqlscan.TokenObjEnd},
				{ID: gqlscan.TokenObjEnd},
			},
			Expect: 9,
		},
		{
			Name: "array_with_tail",
			Input: []gqlparse.Token{
				// []
				{ID: gqlscan.TokenArr},
				{ID: gqlscan.TokenArrEnd},

				// [123]
				{ID: gqlscan.TokenArr},
				{ID: gqlscan.TokenInt, Value: []byte("123")},
				{ID: gqlscan.TokenArrEnd},
			},
			Expect: 2,
		},
		{
			Name: "object_with_tail",
			Input: []gqlparse.Token{
				// {x:"0"}
				{ID: gqlscan.TokenObj},
				{ID: gqlscan.TokenObjField, Value: []byte("x")},
				{ID: gqlscan.TokenStr, Value: []byte("0")},
				{ID: gqlscan.TokenObjEnd},

				// {x:"1"}
				{ID: gqlscan.TokenObj},
				{ID: gqlscan.TokenObjField, Value: []byte("x")},
				{ID: gqlscan.TokenStr, Value: []byte("1")},
				{ID: gqlscan.TokenObjEnd},
			},
			Expect: 4,
		},
	} {
		t.Run(tt.Name, func(t *testing.T) {
			a := scanval.Length(&tokenreader.Reader{Main: tt.Input})
			require.Equal(t, tt.Expect, a)
		})
	}
}
