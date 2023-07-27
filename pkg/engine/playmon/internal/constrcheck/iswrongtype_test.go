package constrcheck

import (
	"testing"

	"github.com/graph-guard/ggproxy/pkg/engine/playmon/internal/tokenreader"
	"github.com/graph-guard/ggproxy/pkg/gqlparse"
	"github.com/graph-guard/gqlscan"
	"github.com/stretchr/testify/require"
	gqlparser "github.com/vektah/gqlparser/v2"
	gqlast "github.com/vektah/gqlparser/v2/ast"
)

func TestIsWrongType_False(t *testing.T) {
	for _, tt := range []struct {
		Name       string
		Schema     string
		GQLVarVals [][]gqlparse.Token
		ExpectType *gqlast.Type
		Input      []gqlparse.Token
	}{
		{
			Name:       "int",
			Schema:     `type Query { x:Int }`,
			ExpectType: &gqlast.Type{NamedType: "Int"},
			Input: []gqlparse.Token{
				{ID: gqlscan.TokenInt, Value: []byte("42")},
			},
		},
		{
			Name:       "float",
			Schema:     `type Query { x:Int }`,
			ExpectType: &gqlast.Type{NamedType: "Float"},
			Input: []gqlparse.Token{
				{ID: gqlscan.TokenFloat, Value: []byte("3.14")},
			},
		},
		{
			Name:       "string",
			Schema:     `type Query { x:Int }`,
			ExpectType: &gqlast.Type{NamedType: "String"},
			Input: []gqlparse.Token{
				{ID: gqlscan.TokenStr, Value: []byte("text")},
			},
		},
		{
			Name:       "string_block",
			Schema:     `type Query { x:Int }`,
			ExpectType: &gqlast.Type{NamedType: "String"},
			Input: []gqlparse.Token{
				{ID: gqlscan.TokenStrBlock},
			},
		},
		{
			Name:       "id",
			Schema:     `type Query { x:Int }`,
			ExpectType: &gqlast.Type{NamedType: "ID"},
			Input: []gqlparse.Token{
				{ID: gqlscan.TokenStr, Value: []byte("someid")},
			},
		},
		{
			Name:       "bool_true",
			Schema:     `type Query { x:Int }`,
			ExpectType: &gqlast.Type{NamedType: "Boolean"},
			Input: []gqlparse.Token{
				{ID: gqlscan.TokenTrue},
			},
		},
		{
			Name:       "bool_false",
			Schema:     `type Query { x:Int }`,
			ExpectType: &gqlast.Type{NamedType: "Boolean"},
			Input: []gqlparse.Token{
				{ID: gqlscan.TokenFalse},
			},
		},
		{
			Name:       "enum",
			Schema:     `enum Color {red}`,
			ExpectType: &gqlast.Type{NamedType: "Color"},
			Input: []gqlparse.Token{
				{ID: gqlscan.TokenEnumVal, Value: []byte("red")},
			},
		},
		{
			Name:       "null",
			Schema:     `type Query { x:Int }`,
			ExpectType: &gqlast.Type{NamedType: "Int", NonNull: false},
			Input: []gqlparse.Token{
				{ID: gqlscan.TokenNull},
			},
		},
		{
			Name:   "array_int",
			Schema: `type Query { x:Int }`,
			ExpectType: &gqlast.Type{Elem: &gqlast.Type{
				NamedType: "Int",
				NonNull:   true,
			}},
			Input: []gqlparse.Token{
				{ID: gqlscan.TokenArr},
				{ID: gqlscan.TokenInt, Value: []byte("0")},
				{ID: gqlscan.TokenInt, Value: []byte("42")},
				{ID: gqlscan.TokenInt, Value: []byte("100500")},
				{ID: gqlscan.TokenArrEnd},
			},
		},
		{
			Name:   "array_int_empty",
			Schema: `type Query { x:Int }`,
			ExpectType: &gqlast.Type{Elem: &gqlast.Type{
				NamedType: "Int",
				NonNull:   true,
			}},
			Input: []gqlparse.Token{
				{ID: gqlscan.TokenArr},
				{ID: gqlscan.TokenArrEnd},
			},
		},
		{
			Name:   "array_int_null",
			Schema: `type Query { x:Int }`,
			ExpectType: &gqlast.Type{Elem: &gqlast.Type{
				NamedType: "Int",
				NonNull:   false,
			}},
			Input: []gqlparse.Token{
				{ID: gqlscan.TokenArr},
				{ID: gqlscan.TokenNull},
				{ID: gqlscan.TokenInt, Value: []byte("42")},
				{ID: gqlscan.TokenNull},
				{ID: gqlscan.TokenInt, Value: []byte("0")},
				{ID: gqlscan.TokenArrEnd},
			},
		},
		{
			Name:   "input_object_1_field",
			Schema: `input InputObject { x: Int! }`,
			ExpectType: &gqlast.Type{
				NamedType: "InputObject",
			},
			Input: []gqlparse.Token{
				// {x:0}
				{ID: gqlscan.TokenObj},
				{ID: gqlscan.TokenObjField, Value: []byte("x")},
				{ID: gqlscan.TokenInt, Value: []byte("0")},
				{ID: gqlscan.TokenObjEnd},
			},
		},
		{
			Name:   "input_object_3_fields",
			Schema: `input InputObject { x: Int! y:String! z:Boolean }`,
			ExpectType: &gqlast.Type{
				NamedType: "InputObject",
			},
			Input: []gqlparse.Token{
				// {x:0, y:"text" z:null}
				{ID: gqlscan.TokenObj},
				{ID: gqlscan.TokenObjField, Value: []byte("x")},
				{ID: gqlscan.TokenInt, Value: []byte("0")},
				{ID: gqlscan.TokenObjField, Value: []byte("y")},
				{ID: gqlscan.TokenStr, Value: []byte("text")},
				{ID: gqlscan.TokenObjField, Value: []byte("z")},
				{ID: gqlscan.TokenNull},
				{ID: gqlscan.TokenObjEnd},
			},
		},
		{
			Name: "nested_input_object",
			Schema: `
				input InputObject { x: O! y:String! z:Boolean }
				input O { i: Int! }
			`,
			ExpectType: &gqlast.Type{
				NamedType: "InputObject",
			},
			Input: []gqlparse.Token{
				// {x:{i:42}, y:"text" z:null}
				{ID: gqlscan.TokenObj},
				{ID: gqlscan.TokenObjField, Value: []byte("x")},
				{ID: gqlscan.TokenObj},
				{ID: gqlscan.TokenObjField, Value: []byte("i")},
				{ID: gqlscan.TokenInt, Value: []byte("42")},
				{ID: gqlscan.TokenObjEnd},
				{ID: gqlscan.TokenObjField, Value: []byte("y")},
				{ID: gqlscan.TokenStr, Value: []byte("text")},
				{ID: gqlscan.TokenObjField, Value: []byte("z")},
				{ID: gqlscan.TokenNull},
				{ID: gqlscan.TokenObjEnd},
			},
		},
		{
			Name: "2d_array_nested_input_objects",
			Schema: `
				input InputObject { x: O! }
				input O { i: Int! }
			`,
			ExpectType: &gqlast.Type{
				Elem: &gqlast.Type{
					Elem: &gqlast.Type{
						NamedType: "InputObject",
					},
				},
			},
			Input: []gqlparse.Token{
				// [[{x:{i:42}}]]
				{ID: gqlscan.TokenArr},
				{ID: gqlscan.TokenArr},
				{ID: gqlscan.TokenObj},
				{ID: gqlscan.TokenObjField, Value: []byte("x")},
				{ID: gqlscan.TokenObj},
				{ID: gqlscan.TokenObjField, Value: []byte("i")},
				{ID: gqlscan.TokenInt, Value: []byte("42")},
				{ID: gqlscan.TokenObjEnd},
				{ID: gqlscan.TokenObjEnd},
				{ID: gqlscan.TokenArrEnd},
				{ID: gqlscan.TokenArrEnd},
			},
		},
		{
			Name:   "custom_scalar_int",
			Schema: `scalar Custom`,
			ExpectType: &gqlast.Type{
				NamedType: "Custom",
			},
			Input: []gqlparse.Token{
				{ID: gqlscan.TokenInt, Value: []byte("0")},
			},
		},
		{
			Name:   "custom_scalar_string",
			Schema: `scalar Custom`,
			ExpectType: &gqlast.Type{
				NamedType: "Custom",
			},
			Input: []gqlparse.Token{
				{ID: gqlscan.TokenStr, Value: []byte("custom")},
			},
		},
		{
			Name:   "custom_scalar_string_block",
			Schema: `scalar Custom`,
			ExpectType: &gqlast.Type{
				NamedType: "Custom",
			},
			Input: []gqlparse.Token{
				{ID: gqlscan.TokenStrBlock},
			},
		},
		{
			Name:   "custom_scalar_enum",
			Schema: `scalar Custom`,
			ExpectType: &gqlast.Type{
				NamedType: "Custom",
			},
			Input: []gqlparse.Token{
				{ID: gqlscan.TokenEnumVal, Value: []byte("customenum")},
			},
		},
		{
			Name:   "custom_scalar_boolean_true",
			Schema: `scalar Custom`,
			ExpectType: &gqlast.Type{
				NamedType: "Custom",
			},
			Input: []gqlparse.Token{
				{ID: gqlscan.TokenTrue},
			},
		},
		{
			Name:   "custom_scalar_boolean_false",
			Schema: `scalar Custom`,
			ExpectType: &gqlast.Type{
				NamedType: "Custom",
			},
			Input: []gqlparse.Token{
				{ID: gqlscan.TokenFalse},
			},
		},
		{
			Name:   "custom_scalar_object",
			Schema: `scalar Custom`,
			ExpectType: &gqlast.Type{
				NamedType: "Custom",
			},
			Input: []gqlparse.Token{
				// {x:{y:[0]}}
				{ID: gqlscan.TokenObj},
				{ID: gqlscan.TokenObjField, Value: []byte("x")},
				{ID: gqlscan.TokenObj},
				{ID: gqlscan.TokenObjField, Value: []byte("y")},
				{ID: gqlscan.TokenArr},
				{ID: gqlscan.TokenInt, Value: []byte("0")},
				{ID: gqlscan.TokenArrEnd},
				{ID: gqlscan.TokenObjEnd},
				{ID: gqlscan.TokenObjEnd},
			},
		},
		{
			Name: "tst",
			Schema: `
				type Query { f(object:Object!):Int }
				input Object { subobject: SubObject! }
				input SubObject { array: [ArrayObject!]! }
				input ArrayObject { name: String!, index: Int! }		  
			`,
			Input: []gqlparse.Token{
				{ID: gqlscan.TokenObj},
				{ID: gqlscan.TokenObjField, Value: []byte("subobject")},
				{ID: gqlscan.TokenObj},
				{ID: gqlscan.TokenObjField, Value: []byte("array")},
				{ID: gqlscan.TokenArr},
				{ID: gqlscan.TokenObj},
				{ID: gqlscan.TokenObjField, Value: []byte("name")},
				{ID: gqlscan.TokenStr, Value: []byte("first")},
				{ID: gqlscan.TokenObjField, Value: []byte("index")},
				{ID: gqlscan.TokenInt, Value: []byte("0")},
				{ID: gqlscan.TokenObjEnd},
				{ID: gqlscan.TokenObj},
				{ID: gqlscan.TokenObjField, Value: []byte("name")},
				{ID: gqlscan.TokenStr, Value: []byte("second")},
				{ID: gqlscan.TokenObjField, Value: []byte("index")},
				{ID: gqlscan.TokenInt, Value: []byte("1")},
				{ID: gqlscan.TokenObjEnd},
				{ID: gqlscan.TokenArrEnd},
				{ID: gqlscan.TokenObjEnd},
				{ID: gqlscan.TokenObjEnd},
			},
		},
		{
			Name:       "expect_float_get_int",
			Schema:     `type Query { x:Int }`,
			ExpectType: &gqlast.Type{NamedType: "Float"},
			Input: []gqlparse.Token{
				{ID: gqlscan.TokenInt, Value: []byte("42")},
			},
		},
		{
			Name:       "expect_float_get_int",
			Schema:     `type Query { x:Int }`,
			ExpectType: &gqlast.Type{NamedType: "Float"},
			GQLVarVals: [][]gqlparse.Token{
				{ /*Not to be used*/ },
				{{ID: gqlscan.TokenInt, Value: []byte("42")}},
			},
			Input: []gqlparse.Token{
				{ID: gqlparse.TokenTypeValIndexOffset + 1},
			},
		},
	} {
		t.Run(tt.Name, func(t *testing.T) {
			var s *gqlast.Schema
			if tt.Schema != "" {
				var err error
				s, err = gqlparser.LoadSchema(&gqlast.Source{
					Name:  "schema.graphqls",
					Input: tt.Schema,
				})
				require.NoError(t, err)
			} else {
				require.Nil(
					t, tt.ExpectType,
					"type expectations are always schema-aware",
				)
			}

			r := isWrongType(
				&tokenreader.Reader{
					Main: tt.Input,
					Vars: tt.GQLVarVals,
				}, tt.ExpectType, s,
			)
			require.False(t, r)
		})
	}
}

func TestIsWrongType_True(t *testing.T) {
	for _, tt := range []struct {
		Name       string
		Schema     string
		GQLVarVals [][]gqlparse.Token
		ExpectType *gqlast.Type
		Input      []gqlparse.Token
	}{
		{
			Name:   "expect_non-null_int_get_null",
			Schema: `type Query { x:Int }`,
			GQLVarVals: [][]gqlparse.Token{
				{ /*Not used*/ },
				{{ID: gqlscan.TokenNull}},
			},
			ExpectType: &gqlast.Type{NamedType: "Int", NonNull: true},
			Input: []gqlparse.Token{
				{ID: gqlparse.TokenTypeValIndexOffset + 1},
			},
		},
		{
			Name:       "expect_non-null_int_get_null",
			Schema:     `type Query { x:Int }`,
			ExpectType: &gqlast.Type{NamedType: "Int", NonNull: true},
			Input: []gqlparse.Token{
				{ID: gqlscan.TokenNull},
			},
		},
		{
			Name:       "expect_int_get_float",
			Schema:     `type Query { x:Int }`,
			ExpectType: &gqlast.Type{NamedType: "Int"},
			Input: []gqlparse.Token{
				{ID: gqlscan.TokenFloat, Value: []byte("3.14")},
			},
		},
		{
			Name: "expect_int_get_enum",
			Schema: `
				type Query { x:Int }
				enum Color { red }
			`,
			ExpectType: &gqlast.Type{NamedType: "Int"},
			Input: []gqlparse.Token{
				{ID: gqlscan.TokenEnumVal, Value: []byte("red")},
			},
		},
		{
			Name: "expect_enum_get_int",
			Schema: `
				type Query { x:Int }
				enum Color { red }
			`,
			ExpectType: &gqlast.Type{NamedType: "Color"},
			Input: []gqlparse.Token{
				{ID: gqlscan.TokenInt, Value: []byte("42")},
			},
		},
		{
			Name:       "expect_boolean_get_enum",
			Schema:     `type Query { x:Int }`,
			ExpectType: &gqlast.Type{NamedType: "Boolean"},
			Input: []gqlparse.Token{
				{ID: gqlscan.TokenEnumVal, Value: []byte("tru")},
			},
		},
		{
			Name:       "expect_enum_get_boolean",
			Schema:     `enum Color { red green blue }`,
			ExpectType: &gqlast.Type{NamedType: "Color"},
			Input: []gqlparse.Token{
				{ID: gqlscan.TokenTrue},
			},
		},
		{
			Name: "expect_enum_get_wrong_enum",
			Schema: `
				enum Color { red green blue }
				enum Fruit { banana orange apple }
			`,
			ExpectType: &gqlast.Type{NamedType: "Color"},
			Input: []gqlparse.Token{
				{ID: gqlscan.TokenEnumVal, Value: []byte("orange")},
			},
		},
		{
			Name:   "expect_object_get_str_block",
			Schema: `input InputObject { x: Int! }`,
			ExpectType: &gqlast.Type{
				NamedType: "InputObject",
			},
			Input: []gqlparse.Token{
				// """not an object"""
				{ID: gqlscan.TokenStrBlock},
			},
		},
		{
			Name:   "expect_different_object_missing_fields",
			Schema: `input InputObject { x: Int! }`,
			ExpectType: &gqlast.Type{
				NamedType: "InputObject",
			},
			Input: []gqlparse.Token{
				// {y:42}
				{ID: gqlscan.TokenObj},
				{ID: gqlscan.TokenObjField, Value: []byte("y")},
				{ID: gqlscan.TokenInt, Value: []byte("42")},
				{ID: gqlscan.TokenObjEnd},
			},
		},
		{
			Name:   "expect_different_object_superfluous_fields",
			Schema: `input InputObject { x: Int! }`,
			ExpectType: &gqlast.Type{
				NamedType: "InputObject",
			},
			Input: []gqlparse.Token{
				// {x:42, y:43}
				{ID: gqlscan.TokenObj},
				{ID: gqlscan.TokenObjField, Value: []byte("x")},
				{ID: gqlscan.TokenInt, Value: []byte("42")},
				{ID: gqlscan.TokenObjField, Value: []byte("y")},
				{ID: gqlscan.TokenInt, Value: []byte("43")},
				{ID: gqlscan.TokenObjEnd},
			},
		},
		{
			Name:   "expect_different_field_type",
			Schema: `input InputObject { x: Int! }`,
			ExpectType: &gqlast.Type{
				NamedType: "InputObject",
			},
			Input: []gqlparse.Token{
				// {x:3.14}
				{ID: gqlscan.TokenObj},
				{ID: gqlscan.TokenObjField, Value: []byte("x")},
				{ID: gqlscan.TokenFloat, Value: []byte("3.14")},
				{ID: gqlscan.TokenObjEnd},
			},
		},
	} {
		t.Run(tt.Name, func(t *testing.T) {
			var s *gqlast.Schema
			if tt.Schema != "" {
				var err error
				s, err = gqlparser.LoadSchema(&gqlast.Source{
					Name:  "schema.graphqls",
					Input: tt.Schema,
				})
				require.NoError(t, err)
			} else {
				require.Nil(
					t, tt.ExpectType,
					"type expectations are always schema-aware",
				)
			}
			r := isWrongType(
				&tokenreader.Reader{
					Main: tt.Input,
					Vars: tt.GQLVarVals,
				}, tt.ExpectType, s,
			)
			require.True(t, r)
		})
	}
}
