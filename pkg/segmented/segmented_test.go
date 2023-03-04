package segmented_test

import (
	"testing"

	"github.com/graph-guard/ggproxy/pkg/segmented"
	"github.com/stretchr/testify/require"
)

func TestCutGet(t *testing.T) {
	s := segmented.New[string, string]()
	require.Equal(t, 0, s.Len())

	s1 := s.Cut("foo")
	require.Equal(t, 1, s.Len())
	require.Equal(t, segmented.Segment{Index: 0, Start: 0, End: 0}, s1)

	s.Append("b1", "bar2", "bar_3")
	s2 := s.Cut("bar")
	require.Equal(t, 2, s.Len())
	require.Equal(t, segmented.Segment{Index: 1, Start: 0, End: 3}, s2)

	s.Append("bz")
	s3 := s.Cut("baz")
	require.Equal(t, 3, s.Len())
	require.Equal(t, segmented.Segment{Index: 2, Start: 3, End: 4}, s3)

	require.Equal(t, s1, s.Get("foo"))
	require.Equal(t, s2, s.Get("bar"))
	require.Equal(t, s3, s.Get("baz"))

	require.Equal(t, []string{}, s.GetSegment(s1))
	require.Equal(t, []string{"b1", "bar2", "bar_3"}, s.GetSegment(s2))
	require.Equal(t, []string{"bz"}, s.GetSegment(s3))

	require.Equal(t, []string{}, s.GetItems("foo"))
	require.Equal(t, []string{"b1", "bar2", "bar_3"}, s.GetItems("bar"))
	require.Equal(t, []string{"bz"}, s.GetItems("baz"))
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
	require.Equal(t, segmented.Segment{Index: -1, Start: 0, End: 0}, x)
}

func TestGetNotFound(t *testing.T) {
	s := segmented.New[string, string]()
	{
		r := s.Get("non-existent")
		require.Equal(t, segmented.Segment{Index: -1}, r)
	}
	s.Append("bar", "baz", "taz")
	s.Cut("foo")
	{
		r := s.Get("non-existent")
		require.Equal(t, segmented.Segment{Index: -1}, r)
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
	s.VisitAll(func(key string, seg segmented.Segment) {
		visitedKeys = append(visitedKeys, key)
		v := s.GetSegment(seg)
		visitedVals = append(visitedVals, v)
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
		const key = "foo"
		require.Nil(t, s.GetItems(key))
		r := s.Get(key)
		require.Equal(t, segmented.Segment{Index: -1}, r)
	}
	{
		const key = "bar"
		require.Nil(t, s.GetItems(key))
		r := s.Get(key)
		require.Equal(t, segmented.Segment{Index: -1}, r)
	}
	{
		const key = "baz"
		require.Nil(t, s.GetItems(key))
		r := s.Get(key)
		require.Equal(t, segmented.Segment{Index: -1}, r)
	}

	s.Append("n1")
	s.Append("n2")
	s.Cut("newkey")
	{
		const key = "newkey"
		require.Equal(t, []string{"n1", "n2"}, s.GetItems(key))
		r := s.Get(key)
		require.Equal(t, 0, r.Index)
		require.Equal(t, segmented.Segment{
			Index: 0, Start: 0, End: 2,
		}, r)
	}
}
