package union_test

import (
	"testing"

	"github.com/graph-guard/constrcheck/internal/token"
	"github.com/graph-guard/constrcheck/internal/union"
	"github.com/graph-guard/gqlscan"
	"github.com/stretchr/testify/require"
)

func TestAreUnionsEqual(t *testing.T) {
	for _, tt := range []struct {
		Name   string
		Left   union.Union
		Right  union.Union
		Expect bool
	}{
		{
			Name:   "same_type_int_equal",
			Left:   union.Int(42),
			Right:  union.Int(42),
			Expect: true,
		},
		{
			Name:   "same_type_int_diff",
			Left:   union.Int(42),
			Right:  union.Int(43),
			Expect: false,
		},

		{
			Name:   "same_type_float_equal",
			Left:   union.Float(3.14),
			Right:  union.Float(3.14),
			Expect: true,
		},
		{
			Name:   "same_type_float_diff",
			Left:   union.Float(3.14),
			Right:  union.Float(3.15),
			Expect: false,
		},

		{
			Name:   "same_type_bool_equal",
			Left:   union.True(),
			Right:  union.True(),
			Expect: true,
		},
		{
			Name:   "same_type_bool_diff",
			Left:   union.True(),
			Right:  union.False(),
			Expect: false,
		},

		{
			Name:   "same_type_string_equal",
			Left:   union.String("okay"),
			Right:  union.String("okay"),
			Expect: true,
		},
		{
			Name:   "same_type_string_diff",
			Left:   union.String("okay"),
			Right:  union.String("!okay"),
			Expect: false,
		},

		{
			Name:   "same_type_enum_equal",
			Left:   union.Enum("red"),
			Right:  union.Enum("red"),
			Expect: true,
		},
		{
			Name:   "same_type_enum_diff",
			Left:   union.Enum("red"),
			Right:  union.Enum("redd"),
			Expect: false,
		},

		{
			Name:   "same_type_null_equal",
			Left:   union.Null(),
			Right:  union.Null(),
			Expect: true,
		},
		{
			Name:   "same_type_null_diff",
			Left:   union.Null(),
			Right:  union.Int(42),
			Expect: false,
		},

		{
			Name: "same_type_tokens_int_equal",
			Left: union.Tokens([]token.Token{
				{Type: gqlscan.TokenInt, Value: []byte("42")},
			}),
			Right: union.Tokens([]token.Token{
				{Type: gqlscan.TokenInt, Value: []byte("42")},
			}),
			Expect: true,
		},
		{
			Name: "same_type_tokens_int_diff",
			Left: union.Tokens([]token.Token{
				{Type: gqlscan.TokenInt, Value: []byte("42")},
			}),
			Right: union.Tokens([]token.Token{
				{Type: gqlscan.TokenInt, Value: []byte("43")},
			}),
			Expect: false,
		},

		{
			Name: "inf_int_tokens",
			Left: union.Int(42),
			Right: union.Tokens([]token.Token{
				{Type: gqlscan.TokenInt, Value: []byte("42")},
			}),
			Expect: true,
		},
		{
			Name: "inf_tokens_int",
			Left: union.Tokens([]token.Token{
				{Type: gqlscan.TokenInt, Value: []byte("42")},
			}),
			Right:  union.Int(42),
			Expect: true,
		},

		{
			Name: "inf_float_tokens",
			Left: union.Float(3.1415),
			Right: union.Tokens([]token.Token{
				{Type: gqlscan.TokenFloat, Value: []byte("3.1415")},
			}),
			Expect: true,
		},
		{
			Name: "inf_tokens_float",
			Left: union.Tokens([]token.Token{
				{Type: gqlscan.TokenFloat, Value: []byte("3.1415")},
			}),
			Right:  union.Float(3.1415),
			Expect: true,
		},

		{
			Name: "inf_string_tokens",
			Left: union.String("okay"),
			Right: union.Tokens([]token.Token{
				{Type: gqlscan.TokenStr, Value: []byte("okay")},
			}),
			Expect: true,
		},
		{
			Name: "inf_tokens_string",
			Left: union.Tokens([]token.Token{
				{Type: gqlscan.TokenStr, Value: []byte("okay")},
			}),
			Right:  union.String("okay"),
			Expect: true,
		},

		{
			Name: "inf_enum_tokens",
			Left: union.Enum("red"),
			Right: union.Tokens([]token.Token{
				{Type: gqlscan.TokenEnumVal, Value: []byte("red")},
			}),
			Expect: true,
		},
		{
			Name: "inf_tokens_enum",
			Left: union.Tokens([]token.Token{
				{Type: gqlscan.TokenEnumVal, Value: []byte("red")},
			}),
			Right:  union.Enum("red"),
			Expect: true,
		},

		{
			Name: "inf_true_tokens",
			Left: union.True(),
			Right: union.Tokens([]token.Token{
				{Type: gqlscan.TokenTrue},
			}),
			Expect: true,
		},
		{
			Name: "inf_tokens_true",
			Left: union.Tokens([]token.Token{
				{Type: gqlscan.TokenTrue},
			}),
			Right:  union.True(),
			Expect: true,
		},

		{
			Name: "inf_false_tokens",
			Left: union.False(),
			Right: union.Tokens([]token.Token{
				{Type: gqlscan.TokenFalse},
			}),
			Expect: true,
		},
		{
			Name: "inf_tokens_false",
			Left: union.Tokens([]token.Token{
				{Type: gqlscan.TokenFalse},
			}),
			Right:  union.False(),
			Expect: true,
		},

		{
			Name: "inf_null_tokens",
			Left: union.Null(),
			Right: union.Tokens([]token.Token{
				{Type: gqlscan.TokenNull},
			}),
			Expect: true,
		},
		{
			Name: "inf_tokens_null",
			Left: union.Tokens([]token.Token{
				{Type: gqlscan.TokenNull},
			}),
			Right:  union.Null(),
			Expect: true,
		},

		{
			Name: "equal_tokens_array",
			Left: union.Tokens([]token.Token{
				{Type: gqlscan.TokenArr},
				{Type: gqlscan.TokenInt, Value: []byte("1")},
				{Type: gqlscan.TokenInt, Value: []byte("23")},
				{Type: gqlscan.TokenArr},
			}),
			Right: union.Tokens([]token.Token{
				{Type: gqlscan.TokenArr},
				{Type: gqlscan.TokenInt, Value: []byte("1")},
				{Type: gqlscan.TokenInt, Value: []byte("23")},
				{Type: gqlscan.TokenArr},
			}),
			Expect: true,
		},
		{
			Name: "not_equal_tokens_array",
			Left: union.Tokens([]token.Token{
				{Type: gqlscan.TokenArr},
				{Type: gqlscan.TokenInt, Value: []byte("1")},
				{Type: gqlscan.TokenInt, Value: []byte("23")},
				{Type: gqlscan.TokenArr},
			}),
			Right: union.Tokens([]token.Token{
				{Type: gqlscan.TokenArr},
				{Type: gqlscan.TokenInt, Value: []byte("23")},
				{Type: gqlscan.TokenInt, Value: []byte("1")},
				{Type: gqlscan.TokenArr},
			}),
			Expect: false,
		},
		{
			Name: "not_equal_tokens_array_empty",
			Left: union.Tokens([]token.Token{
				{Type: gqlscan.TokenArr},
				{Type: gqlscan.TokenInt, Value: []byte("1")},
				{Type: gqlscan.TokenInt, Value: []byte("23")},
				{Type: gqlscan.TokenArr},
			}),
			Right: union.Tokens([]token.Token{
				{Type: gqlscan.TokenArr},
				{Type: gqlscan.TokenArr},
			}),
			Expect: false,
		},

		{
			Name: "inf_tokens_null",
			Left: union.Tokens([]token.Token{
				{Type: gqlscan.TokenNull},
			}),
			Right:  union.Null(),
			Expect: true,
		},

		{
			Name:   "diff_type_int_float",
			Left:   union.Int(42),
			Right:  union.Float(42),
			Expect: false,
		},
	} {
		t.Run(tt.Name, func(t *testing.T) {
			require.Equal(t, tt.Expect, union.Equal(tt.Left, tt.Right))
		})
	}
}
