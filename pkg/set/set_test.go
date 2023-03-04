package set_test

import (
	"testing"

	"github.com/graph-guard/ggproxy/pkg/set"

	"github.com/stretchr/testify/require"
)

func TestEnable(t *testing.T) {
	s := set.New("a", "b", "c")
	Expect(t, s)

	require.True(t, s.Enable("b"))
	Expect(t, s, "b")

	require.True(t, s.Enable("c"))
	Expect(t, s, "b", "c")

	require.True(t, s.Enable("a"))
	Expect(t, s, "b", "c", "a")
}

func TestDisable(t *testing.T) {
	s := set.New("a", "b", "c")
	for _, b := range []string{"c", "b", "a"} {
		require.True(t, s.Enable(b))
	}
	Expect(t, s, "c", "b", "a")

	require.True(t, s.Disable("b"))
	Expect(t, s, "c", "a")

	require.True(t, s.Disable("c"))
	Expect(t, s, "a")

	require.True(t, s.Disable("a"))
	Expect(t, s)
}

func TestEnableUnknownNoop(t *testing.T) {
	s := set.New("a", "b", "c")
	Expect(t, s)
	require.False(t, s.Enable("d"))
	Expect(t, s)
}

func TestDisableUnknownNoop(t *testing.T) {
	s := set.New("a", "b", "c")
	Expect(t, s)
	for _, b := range []string{"a", "b", "c"} {
		require.True(t, s.Enable(b))
	}
	Expect(t, s, "a", "b", "c")
	require.False(t, s.Disable("d"))
	Expect(t, s, "a", "b", "c")
}

func TestEmptyNoop(t *testing.T) {
	s := set.New[string]()
	Expect(t, s)
	require.False(t, s.Enable("a"))
	Expect(t, s)
	require.False(t, s.Disable("a"))
	Expect(t, s)
}

func TestReset(t *testing.T) {
	s := set.New("a", "b", "c")
	Expect(t, s)
	for _, b := range []string{"a", "b", "c"} {
		require.True(t, s.Enable(b))
	}
	Expect(t, s, "a", "b", "c")
	s.Reset()
	Expect(t, s)
}

func TestAdd(t *testing.T) {
	s := set.New("a", "b", "c")
	Expect(t, s)
	for _, b := range []string{"a", "b"} {
		require.True(t, s.Enable(b))
	}
	Expect(t, s, "a", "b")

	require.True(t, s.Add("d"))
	Expect(t, s, "a", "b")

	require.True(t, s.Enable("d"))
	Expect(t, s, "a", "b", "d")
}

func TestAddKnownNoop(t *testing.T) {
	s := set.New("a", "b", "c")
	Expect(t, s)
	for _, b := range []string{"a", "b"} {
		require.True(t, s.Enable(b))
	}
	Expect(t, s, "a", "b")

	require.False(t, s.Add("b"))
	Expect(t, s, "a", "b")
	require.False(t, s.Add("c"))
	Expect(t, s, "a", "b")
}

func Expect[T comparable](t *testing.T, s *set.Set[T], e ...T) {
	t.Helper()
	var enabled []T
	require.Equal(t, len(e), s.Enabled())
	s.VisitEnabled(func(t T) (stop bool) {
		enabled = append(enabled, t)
		return false
	})
	require.Equal(t, e, enabled)
}
