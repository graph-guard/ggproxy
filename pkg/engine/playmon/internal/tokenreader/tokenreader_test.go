package tokenreader_test

import (
	"testing"

	"github.com/graph-guard/ggproxy/pkg/engine/playmon/internal/tokenreader"
	"github.com/graph-guard/ggproxy/pkg/gqlparse"
	"github.com/graph-guard/gqlscan"
	"github.com/stretchr/testify/require"
)

// No need to test the case when Main is exhausted, it will never happen.

func TestReadOne(t *testing.T) {
	r := &tokenreader.Reader{
		Vars: [][]gqlparse.Token{
			{T(gqlscan.TokenStr, "var1_first"), T(gqlscan.TokenStr, "var1_second")},
			{T(gqlscan.TokenStr, "var2_first"), T(gqlscan.TokenStr, "var2_second")},
		},
		Main: []gqlparse.Token{
			T(gqlparse.TokenTypeValIndexOffset+1, "irrelevant_text"),
			T(gqlscan.TokenStrBlock, "intermediate"),
			T(gqlparse.TokenTypeValIndexOffset, "irrelevant_text2"),
			T(gqlscan.TokenStrBlock, "last_main"),
		},
	}

	require.False(t, r.EOF())
	TokEq(t, T(gqlscan.TokenStr, "var2_first"), r.PeekOne())
	TokEq(t, T(gqlscan.TokenStr, "var2_first"), r.ReadOne())
	require.False(t, r.EOF())
	TokEq(t, T(gqlscan.TokenStr, "var2_second"), r.PeekOne())
	TokEq(t, T(gqlscan.TokenStr, "var2_second"), r.ReadOne())
	require.False(t, r.EOF())
	TokEq(t, T(gqlscan.TokenStrBlock, "intermediate"), r.PeekOne())
	TokEq(t, T(gqlscan.TokenStrBlock, "intermediate"), r.ReadOne())
	require.False(t, r.EOF())
	TokEq(t, T(gqlscan.TokenStr, "var1_first"), r.PeekOne())
	TokEq(t, T(gqlscan.TokenStr, "var1_first"), r.ReadOne())
	require.False(t, r.EOF())
	TokEq(t, T(gqlscan.TokenStr, "var1_second"), r.PeekOne())
	TokEq(t, T(gqlscan.TokenStr, "var1_second"), r.ReadOne())
	require.False(t, r.EOF())
	TokEq(t, T(gqlscan.TokenStrBlock, "last_main"), r.PeekOne())
	TokEq(t, T(gqlscan.TokenStrBlock, "last_main"), r.ReadOne())
	require.True(t, r.EOF())
	require.True(t, r.EOF())
}

func TestSkipUntil(t *testing.T) {
	r := &tokenreader.Reader{
		Vars: [][]gqlparse.Token{
			{T(gqlscan.TokenStr, "var1_first"), T(gqlscan.TokenStr, "var1_second")},
			{T(gqlscan.TokenStr, "var2_first"), T(gqlscan.TokenStr, "var2_second")},
		},
		Main: []gqlparse.Token{
			T(gqlparse.TokenTypeValIndexOffset+1, "irrelevant_text"),
			T(gqlscan.TokenStrBlock, "intermediate"),
			T(gqlparse.TokenTypeValIndexOffset, "irrelevant_text2"),
			T(gqlscan.TokenArrEnd, "last_main"),
		},
	}

	r.SkipUntil(gqlscan.TokenArrEnd)
	require.Equal(t, &tokenreader.Reader{
		Vars: [][]gqlparse.Token{
			{T(gqlscan.TokenStr, "var1_first"), T(gqlscan.TokenStr, "var1_second")},
			{T(gqlscan.TokenStr, "var2_first"), T(gqlscan.TokenStr, "var2_second")},
		},
		Var:  []gqlparse.Token{},
		Main: []gqlparse.Token{},
	}, r)
}

func TestSkipUntil_InVarVal(t *testing.T) {
	r := &tokenreader.Reader{
		Vars: [][]gqlparse.Token{
			{T(gqlscan.TokenStr, "var1_first"), T(gqlscan.TokenStr, "var1_second")},
			{T(gqlscan.TokenEnumVal, "var2_first"), T(gqlscan.TokenStr, "var2_second")},
		},
		Main: []gqlparse.Token{
			T(gqlparse.TokenTypeValIndexOffset+1, "irrelevant_text"),
			T(gqlscan.TokenStrBlock, "intermediate"),
			T(gqlparse.TokenTypeValIndexOffset, "irrelevant_text2"),
			T(gqlscan.TokenArrEnd, "last_main"),
		},
	}

	r.SkipUntil(gqlscan.TokenEnumVal)
	require.Equal(t, &tokenreader.Reader{
		Vars: [][]gqlparse.Token{
			{T(gqlscan.TokenStr, "var1_first"), T(gqlscan.TokenStr, "var1_second")},
			{T(gqlscan.TokenEnumVal, "var2_first"), T(gqlscan.TokenStr, "var2_second")},
		},
		Var: []gqlparse.Token{
			T(gqlscan.TokenStr, "var2_second"),
		},
		Main: []gqlparse.Token{
			T(gqlscan.TokenStrBlock, "intermediate"),
			T(gqlparse.TokenTypeValIndexOffset, "irrelevant_text2"),
			T(gqlscan.TokenArrEnd, "last_main"),
		},
	}, r)
}

func TokEq(t *testing.T, expect, actual gqlparse.Token) {
	t.Helper()
	require.Equal(t, expect.String(), actual.String())
}

func T(id gqlscan.Token, value string) gqlparse.Token {
	var v []byte
	if value != "" {
		v = []byte(value)
	}
	return gqlparse.Token{ID: gqlscan.Token(id), Value: v}
}

var GT gqlparse.Token

func BenchmarkReadOne(b *testing.B) {
	r := &tokenreader.Reader{
		Vars: [][]gqlparse.Token{
			{T(gqlscan.TokenStr, "var1_first"), T(gqlscan.TokenStr, "var1_second")},
			{T(gqlscan.TokenStr, "var2_first"), T(gqlscan.TokenStr, "var2_second")},
		},
		Main: []gqlparse.Token{
			T(gqlparse.TokenTypeValIndexOffset+1, "irrelevant_text"),
			T(gqlscan.TokenStrBlock, "intermediate"),
			T(gqlparse.TokenTypeValIndexOffset, "irrelevant_text2"),
			T(gqlscan.TokenStrBlock, "last_main"),
		},
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		resetTo := *r
		GT = r.ReadOne() // var2_first
		GT = r.ReadOne() // var2_second
		GT = r.ReadOne() // intermediate
		GT = r.ReadOne() // var1_first
		GT = r.ReadOne() // var1_second
		GT = r.ReadOne() // last_main
		*r = resetTo     // Reset reader
	}
}

func BenchmarkSkipUntil(b *testing.B) {
	r := &tokenreader.Reader{
		Vars: [][]gqlparse.Token{
			{T(gqlscan.TokenStr, "var1_first"), T(gqlscan.TokenStr, "var1_second")},
			{T(gqlscan.TokenStr, "var2_first"), T(gqlscan.TokenStr, "var2_second")},
		},
		Main: []gqlparse.Token{
			T(gqlparse.TokenTypeValIndexOffset+1, "irrelevant_text"),
			T(gqlscan.TokenStrBlock, "intermediate"),
			T(gqlparse.TokenTypeValIndexOffset, "irrelevant_text2"),
			T(gqlscan.TokenArrEnd, "last_main"),
		},
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		resetTo := *r
		r.SkipUntil(gqlscan.TokenArrEnd)
		*r = resetTo // Reset reader
	}
}
