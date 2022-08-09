package segmented_test

import (
	"testing"

	"github.com/graph-guard/gguard-proxy/utilities/segmented"
	"github.com/stretchr/testify/require"
)

func TestCutGet(t *testing.T) {
	s := segmented.New[string, string]()
	require.Equal(t, 0, s.Len())

	s1 := s.Cut("foo")
	require.Equal(t, 1, s.Len())
	require.Equal(t, segmented.Segment{Start: 0, End: 0}, s1)

	s.Append("b1", "bar2", "bar_3")
	s2 := s.Cut("bar")
	require.Equal(t, 2, s.Len())
	require.Equal(t, segmented.Segment{Start: 0, End: 3}, s2)

	s.Append("bz")
	s3 := s.Cut("baz")
	require.Equal(t, 3, s.Len())
	require.Equal(t, segmented.Segment{Start: 3, End: 4}, s3)

	require.Equal(t, []string{}, s.Get("foo"))
	require.Equal(t, []string{"b1", "bar2", "bar_3"}, s.Get("bar"))
	require.Equal(t, []string{"bz"}, s.Get("baz"))

	require.Equal(t, []string{}, s.GetSegment(s1))
	require.Equal(t, []string{"b1", "bar2", "bar_3"}, s.GetSegment(s2))
	require.Equal(t, []string{"bz"}, s.GetSegment(s3))
}

func TestCutExists(t *testing.T) {
	s := segmented.New[string, string]()

	s.Append("f1", "f2")
	s.Cut("foo")
	s.Append("b1")
	s.Cut("bar")
	s.Cut("baz")

	require.Equal(t, 3, s.Len())
	x := s.Cut("foo")
	require.Equal(t, 3, s.Len())
	require.Equal(t, segmented.Segment{Start: -1, End: 0}, x)
}

func TestGetNotFound(t *testing.T) {
	s := segmented.New[string, string]()
	{
		r := s.Get("non-existent")
		require.Nil(t, r)
	}
	s.Append("bar", "baz", "taz")
	s.Cut("foo")
	{
		r := s.Get("non-existent")
		require.Nil(t, r)
	}
}

func TestVisitAll(t *testing.T) {
	s := segmented.New[string, string]()

	s.Append("f1", "f2")
	s.Cut("foo")
	s.Append("b1")
	s.Cut("bar")
	s.Cut("baz")

	visitedKeys := []string{}
	visitedVals := [][]string{}
	s.VisitAll(func(key string, s []string) {
		visitedKeys = append(visitedKeys, key)
		visitedVals = append(visitedVals, s)
	})

	require.Len(t, visitedKeys, 3)

	for _, x := range [3]string{"foo", "bar", "baz"} {
		for i := range visitedKeys {
			if visitedKeys[i] != x {
				continue
			}
		}
	}

	require.Contains(t, visitedKeys, "foo")
	require.Contains(t, visitedKeys, "bar")
	require.Contains(t, visitedKeys, "baz")
}

func TestReset(t *testing.T) {
	s := segmented.New[string, string]()

	s.Append("f1", "f2")
	s.Cut("foo")
	s.Append("b1")
	s.Cut("bar")
	s.Cut("baz")

	require.Equal(t, 3, s.Len())

	s.Reset()
	require.Equal(t, 0, s.Len())
	{
		r := s.Get("foo")
		require.Nil(t, r)
	}
	{
		r := s.Get("bar")
		require.Nil(t, r)
	}
	{
		r := s.Get("baz")
		require.Nil(t, r)
	}
}
