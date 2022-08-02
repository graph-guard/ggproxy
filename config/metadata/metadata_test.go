package metadata_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/graph-guard/gguard-proxy/config/metadata"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	in := lines(
		"---",
		"name: Okay",
		"tags:",
		"  - foo",
		"  - bar",
		"---",
		"body",
	)
	m, body, err := metadata.Parse(in)
	require.NoError(t, err)
	require.Equal(t, metadata.Metadata{
		Name: "Okay",
		Tags: []string{"foo", "bar"},
	}, m)
	require.Equal(t, "body\n", string(body))
}

func TestParseNoMetadata(t *testing.T) {
	in := lines(
		"one",
		"two",
		"three",
	)
	m, body, err := metadata.Parse(in)
	require.NoError(t, err)
	require.Zero(t, m)
	require.Equal(t, string(lines(
		"one",
		"two",
		"three",
	)), string(body))
}

func TestParseSpaces(t *testing.T) {
	in := lines(
		" \t\r",
		"--- \t\r",
		"name: Okay",
		"tags:",
		"  - foo",
		"  - bar",
		"--- \t\r",
		"body",
	)
	m, body, err := metadata.Parse(in)
	require.NoError(t, err)
	require.Equal(t, metadata.Metadata{
		Name: "Okay",
		Tags: []string{"foo", "bar"},
	}, m)
	require.Equal(t, "body\n", string(body))
}

func TestParseEmptyInput(t *testing.T) {
	m, body, err := metadata.Parse([]byte(""))
	require.NoError(t, err)
	require.Zero(t, m)
	require.Equal(t, "", string(body))
}

func TestParseHeaderOnly(t *testing.T) {
	m, body, err := metadata.Parse(lines(
		"---",
		"name: X",
		"---",
	))
	require.NoError(t, err)
	require.Equal(t, metadata.Metadata{
		Name: "X",
	}, m)
	require.Equal(t, "", string(body))
}

func TestParseErrMalformedInitializer(t *testing.T) {
	m, body, err := metadata.Parse(lines(
		"---a",
		"name: X",
	))
	require.Error(t, err)
	require.True(t, errors.Is(err, metadata.ErrExpectedDelimiter))
	require.Zero(t, m)
	require.Equal(t, "", string(body))
}

func TestParseErrMalformedClosingDelimiter(t *testing.T) {
	m, body, err := metadata.Parse(lines(
		"---",
		"name: X",
		"----",
		"body",
	))
	require.Error(t, err)
	require.True(t, errors.Is(err, metadata.ErrExpectedDelimiter))
	require.Zero(t, m)
	require.Equal(t, "", string(body))
}

func TestParseErrMalformedMetadata(t *testing.T) {
	m, body, err := metadata.Parse(lines(
		"---",
		"unknownField: 42",
		"---",
		"body",
	))
	require.Error(t, err)
	require.Zero(t, m)
	require.Equal(t, "body\n", string(body))
}

func TestParseErrUnexpectedEOF(t *testing.T) {
	m, body, err := metadata.Parse([]byte("---"))
	require.Error(t, err)
	require.Zero(t, m)
	require.Equal(t, "", string(body))
}

func TestParseErrMissingClosingDelimiter(t *testing.T) {
	m, body, err := metadata.Parse(lines(
		"---",
		"Name: x",
		"   ",
	))
	require.Error(t, err)
	require.Zero(t, m)
	require.Equal(t, "", string(body))
}

func lines(lines ...string) []byte {
	var b strings.Builder
	for i := range lines {
		b.WriteString(lines[i])
		b.WriteByte('\n')
	}
	return []byte(b.String())
}
