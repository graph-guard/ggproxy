package tokenwriter_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/graph-guard/gguard-proxy/gqlreduce"
	"github.com/graph-guard/gguard-proxy/server/internal/tokenwriter"
	"github.com/graph-guard/gqlscan"
	"github.com/stretchr/testify/require"
)

func TestWrite(t *testing.T) {
	for _, td := range []struct {
		Request       string
		OperationName string
		Expect        string
	}{
		{
			Request: `{foo}`,
			Expect:  `{foo}`,
		},
		{
			Request: `query {foo bar}`,
			Expect:  `{foo bar}`,
		},
		{
			Request: `mutation {foo bar}`,
			Expect:  `mutation {foo bar}`,
		},
		{
			Request: `subscription {foo bar}`,
			Expect:  `subscription {foo bar}`,
		},
		{
			Request:       `query OprName {foo bar}`,
			Expect:        `query OprName {foo bar}`,
			OperationName: "OprName",
		},
		{
			Request:       `mutation OprName {foo bar}`,
			Expect:        `mutation OprName {foo bar}`,
			OperationName: "OprName",
		},
		{
			Request:       `subscription OprName {foo bar}`,
			Expect:        `subscription OprName {foo bar}`,
			OperationName: "OprName",
		},
		{
			Request: `{foo bar}`,
			Expect:  `{foo bar}`,
		},
		{
			Request: `{foo bar baz}`,
			Expect:  `{foo bar baz}`,
		},
		{
			Request: `{alias:foo}`,
			Expect:  `{alias:foo}`,
		},
		{
			Request: `{aliasA:foo aliasB:bar}`,
			Expect:  `{aliasA:foo aliasB:bar}`,
		},
		{
			Request: `{...on T{foo}}`,
			Expect:  `{...on T{foo}}`,
		},
		{
			Request: `{...on A{foo} ...on B{bar}}`,
			Expect:  `{...on A{foo} ...on B{bar}}`,
		},
		{
			Request: `{foo @directive ...on B{bar}}`,
			Expect:  `{foo @directive ...on B{bar}}`,
		},
		{
			Request: `{foo(a:42)}`,
			Expect:  `{foo(a:42)}`,
		},
		{
			Request: `{foo(a:42) bar}`,
			Expect:  `{foo(a:42) bar}`,
		},
		{
			Request: `{foo(a:42) ...on B{bar}}`,
			Expect:  `{foo(a:42) ...on B{bar}}`,
		},
		{
			Request: `{foo(a:42) alias:bar}`,
			Expect:  `{foo(a:42) alias:bar}`,
		},
		{
			Request: `{foo(a:42 b:"value")}`,
			Expect:  `{foo(a:42 b:"value")}`,
		},
		{
			Request: `{foo(a:42 b:"value" c:[[null false true] []])}`,
			Expect:  `{foo(a:42 b:"value" c:[[null false true] []])}`,
		},
		{
			Request: `{foo(x:{o:[[null false true] []] o2:"""okay"""})}`,
			Expect:  `{foo(x:{o:[[null false true] []] o2:"""okay"""})}`,
		},
		{
			Request: `{foo @directive bar @directive}`,
			Expect:  `{foo @directive bar @directive}`,
		},
		{
			Request: `{foo(x:0) @directive bar(x:0) @directive}`,
			Expect:  `{foo(x:0) @directive bar(x:0) @directive}`,
		},
		{
			Request: `{foo(x:[{f:2} {f:3} {f:4}])}`,
			Expect:  `{foo(x:[{f:2} {f:3} {f:4}])}`,
		},
		{
			Request: `{foo(x:[[[]] [[]] [[]]])}`,
			Expect:  `{foo(x:[[[]] [[]] [[]]])}`,
		},
		{
			Request: `{foo(x:[null null null])}`,
			Expect:  `{foo(x:[null null null])}`,
		},
		{
			Request: `{foo(x:[true true true])}`,
			Expect:  `{foo(x:[true true true])}`,
		},
		{
			Request: `{foo(x:[false false false])}`,
			Expect:  `{foo(x:[false false false])}`,
		},
		{
			Request: `{foo(x:[42 42 42])}`,
			Expect:  `{foo(x:[42 42 42])}`,
		},
		{
			Request: `{foo(x:[42.5 42.5 42.5])}`,
			Expect:  `{foo(x:[42.5 42.5 42.5])}`,
		},
		{
			Request: `{foo(x:["string" "string" "string"])}`,
			Expect:  `{foo(x:["string" "string" "string"])}`,
		},
		{
			Request: `{foo(x:["""string""" """string""" """string"""])}`,
			Expect:  `{foo(x:["""string""" """string""" """string"""])}`,
		},
		{
			Request: `{foo(x:[EnumValue EnumValue EnumValue])}`,
			Expect:  `{foo(x:[EnumValue EnumValue EnumValue])}`,
		},
	} {
		t.Run("", func(t *testing.T) {
			r := gqlreduce.NewReducer()
			r.Reduce(
				[]byte(td.Request),
				[]byte(td.OperationName),
				nil,
				func(operation []gqlreduce.Token) {
					var b bytes.Buffer
					require.NoError(t, tokenwriter.Write(&b, operation))
					require.Equal(t, td.Expect, b.String())
				},
				func(err error) {
					t.Fatalf("unexpected error: %v", err)
				},
			)
		})
	}
}

func TestEmptyInput(t *testing.T) {
	var b bytes.Buffer
	err := tokenwriter.Write(&b, []gqlreduce.Token{})
	require.NoError(t, err)
	require.Equal(t, "", b.String())
}

func TestErr(t *testing.T) {
	tokens := Parse(t, `query($v:Int=42) {foo(a:$v)}`)
	var b bytes.Buffer
	err := tokenwriter.Write(&b, tokens)
	require.Error(t, err)
	require.Equal(t, "unsupported token type: variable list", err.Error())
}

func TestWriterErr(t *testing.T) {
	tokens := Parse(t, `{foo}`)
	w := &Writer{Responses: []WriterResponse{
		{0, ErrWriter},
	}}
	err := tokenwriter.Write(w, tokens)
	require.Error(t, err)
	require.Equal(t, ErrWriter, err)
}

type Writer struct {
	Responses []WriterResponse
}

type WriterResponse struct {
	N   int
	Err error
}

var ErrWriter = errors.New("writer error")

func (w *Writer) Write(data []byte) (int, error) {
	if len(w.Responses) < 1 {
		panic("no responses left")
	}
	r := w.Responses[0]
	w.Responses = w.Responses[1:]
	n := len(data)
	if r.N > -1 {
		n = r.N
	}
	return n, r.Err
}

func Parse(t *testing.T, src string) (tokens []gqlreduce.Token) {
	err := gqlscan.ScanAll(
		[]byte(src),
		func(i *gqlscan.Iterator) {
			tokens = append(tokens, gqlreduce.Token{
				Type:  i.Token(),
				Value: i.Value(),
			})
		},
	)
	require.False(t, err.IsErr())
	return tokens
}
