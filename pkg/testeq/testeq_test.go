package testeq_test

import (
	"fmt"
	"testing"

	"github.com/graph-guard/ggproxy/pkg/testeq"
	"github.com/stretchr/testify/require"
)

func TestMapsEmpty(t *testing.T) {
	w := new(TestWriter)
	exp, act := map[string]string{}, map[string]string{}
	ok := testeq.Maps(w, "key", exp, act, compareStrings, stringify)
	require.Len(t, w.Writes, 0)
	require.True(t, ok)
}

func TestMapsEqual(t *testing.T) {
	w := new(TestWriter)
	exp, act := map[string]string{
		"a": "1",
		"b": "2",
		"c": "3",
	}, map[string]string{
		"a": "1",
		"b": "2",
		"c": "3",
	}
	ok := testeq.Maps(w, "key", exp, act, compareStrings, stringify)
	require.Len(t, w.Writes, 0)
	require.True(t, ok)
}

func TestMapsMismatchOne(t *testing.T) {
	w := new(TestWriter)
	exp, act := map[string]string{
		"a": "1",
		"b": "2",
		"c": "3",
	}, map[string]string{
		"a": "1",
		"b": "2",
		"c": "x",
	}
	ok := testeq.Maps(w, "key", exp, act, compareStrings, stringify)
	require.Equal(t, []string{
		"mismatching key c: not equal",
	}, w.Writes)
	require.False(t, ok)
}

func TestMapsMismatchMultiple(t *testing.T) {
	w := new(TestWriter)
	exp, act := map[string]string{
		"a": "1",
		"b": "2",
		"c": "3",
	}, map[string]string{
		"a": "y",
		"b": "2",
		"c": "x",
	}
	ok := testeq.Maps(w, "key", exp, act, compareStrings, stringify)
	require.Equal(t, []string{
		"mismatching key a: not equal",
		"mismatching key c: not equal",
	}, w.Writes)
	require.False(t, ok)
}

func TestMapsMissing(t *testing.T) {
	w := new(TestWriter)
	exp, act := map[string]string{
		"a": "1",
		"b": "2",
		"c": "3",
		"d": "4",
	}, map[string]string{
		"b": "2",
	}
	ok := testeq.Maps(w, "key", exp, act, compareStrings, stringify)
	require.Equal(t, []string{
		"missing key a (1)",
		"missing key c (3)",
		"missing key d (4)",
	}, w.Writes)
	require.False(t, ok)
}

func TestMapsUnexpected(t *testing.T) {
	w := new(TestWriter)
	exp, act := map[string]string{
		"b": "2",
	}, map[string]string{
		"a": "1",
		"b": "2",
		"c": "3",
		"d": "4",
	}
	ok := testeq.Maps(w, "key", exp, act, compareStrings, stringify)
	require.Equal(t, []string{
		"unexpected key a (1)",
		"unexpected key c (3)",
		"unexpected key d (4)",
	}, w.Writes)
	require.False(t, ok)
}

func TestSlicesEmpty(t *testing.T) {
	w := new(TestWriter)
	exp, act := []string{}, []string{}
	ok := testeq.Slices(w, "key", exp, act, compareStrings, stringify)
	require.Len(t, w.Writes, 0)
	require.True(t, ok)
}

func TestSlicesEqual(t *testing.T) {
	w := new(TestWriter)
	exp, act := []string{"a", "b", "c"}, []string{"a", "b", "c"}
	ok := testeq.Slices(w, "key", exp, act, compareStrings, stringify)
	require.Len(t, w.Writes, 0)
	require.True(t, ok)
}

func TestSlicesMismatchOne(t *testing.T) {
	w := new(TestWriter)
	exp, act := []string{"a", "b", "c"}, []string{"a", "b", "x"}
	ok := testeq.Slices(w, "key", exp, act, compareStrings, stringify)
	require.Equal(t, []string{
		"mismatching key at index 2: not equal",
	}, w.Writes)
	require.False(t, ok)
}

func TestSlicesMismatchMultiple(t *testing.T) {
	w := new(TestWriter)
	exp, act := []string{"a", "b", "c"}, []string{"y", "b", "x"}
	ok := testeq.Slices(w, "key", exp, act, compareStrings, stringify)
	require.Equal(t, []string{
		"mismatching key at index 0: not equal",
		"mismatching key at index 2: not equal",
	}, w.Writes)
	require.False(t, ok)
}

func TestSlicesMissing(t *testing.T) {
	w := new(TestWriter)
	exp, act := []string{"a", "b", "c", "d"}, []string{"a"}
	ok := testeq.Slices(w, "key", exp, act, compareStrings, stringify)
	require.Equal(t, []string{
		"missing key at index 1 (b)",
		"missing key at index 2 (c)",
		"missing key at index 3 (d)",
	}, w.Writes)
	require.False(t, ok)
}

func TestSlicesUnexpected(t *testing.T) {
	w := new(TestWriter)
	exp, act := []string{"a"}, []string{"a", "b", "c", "d"}
	ok := testeq.Slices(w, "key", exp, act, compareStrings, stringify)
	require.Equal(t, []string{
		"unexpected key at index 1 (b)",
		"unexpected key at index 2 (c)",
		"unexpected key at index 3 (d)",
	}, w.Writes)
	require.False(t, ok)
}

type TestWriter struct{ Writes []string }

func (t *TestWriter) Errorf(format string, v ...any) {
	t.Writes = append(t.Writes, fmt.Sprintf(format, v...))
}
func (t *TestWriter) Helper() {}

func compareStrings(exp, act string) (errMsg string) {
	if exp != act {
		return "not equal"
	}
	return ""
}

func stringify(s string) string { return s }
