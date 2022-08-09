package container_test

import (
	"strconv"
	"testing"

	"github.com/graph-guard/gguard-proxy/utilities/container"
	"github.com/graph-guard/gguard-proxy/utilities/container/gomap"
	"github.com/graph-guard/gguard-proxy/utilities/container/hamap"
	"github.com/graph-guard/gguard-proxy/utilities/container/linear"
	"github.com/stretchr/testify/require"
)

var implementations = []struct {
	Name string
	Make func(capacity int) container.Mapper[[]byte, int]
}{
	{"gomap", func(capacity int) container.Mapper[[]byte, int] {
		return gomap.New[[]byte, int](capacity)
	}},
	{"linear", func(capacity int) container.Mapper[[]byte, int] {
		return linear.New[[]byte, int](capacity)
	}},
	{"hamap", func(capacity int) container.Mapper[[]byte, int] {
		return hamap.New[[]byte, int](capacity, nil)
	}},
}

func forEachImplT(
	t *testing.T,
	fn func(*testing.T, container.Mapper[[]byte, int]),
) {
	for _, impl := range implementations {
		t.Run(impl.Name, func(t *testing.T) {
			fn(t, impl.Make(0))
		})
	}
}

func TestReset(t *testing.T) {
	forEachImplT(t, func(t *testing.T, m container.Mapper[[]byte, int]) {
		numKeys := 5
		for i := 0; i < numKeys; i++ {
			m.Set([]byte(strconv.Itoa(i)), i)
		}
		require.Equal(t, numKeys, m.Len())

		m.Reset()

		require.Zero(t, m.Len())
		for i := 0; i < numKeys; i++ {
			v, ok := m.Get([]byte(strconv.Itoa(i)))
			require.Zero(t, v)
			require.False(t, ok)
		}
	})
}

func TestSet(t *testing.T) {
	forEachImplT(t, func(t *testing.T, m container.Mapper[[]byte, int]) {
		m.Set([]byte("a"), -1)
		m.Set([]byte("b"), 0)
		m.Set([]byte("c"), 1)
		Expect(t, m, map[string]int{
			"a": -1,
			"b": 0,
			"c": 1,
		})
		m.Set([]byte("a"), 2)
		m.Set([]byte("b"), 3)
		m.Set([]byte("c"), 4)
		Expect(t, m, map[string]int{
			"a": 2,
			"b": 3,
			"c": 4,
		})
	})
}

func TestGet(t *testing.T) {
	forEachImplT(t, func(t *testing.T, m container.Mapper[[]byte, int]) {
		m.Set([]byte("a"), 2)
		m.Set([]byte("b"), 3)

		HasVal(t, m, []byte("b"), 3)

		v, ok := m.Get([]byte("nonexistent"))
		require.False(t, ok)
		require.Zero(t, v)
	})
}

func TestDelete(t *testing.T) {
	forEachImplT(t, func(t *testing.T, m container.Mapper[[]byte, int]) {
		m.Set([]byte("a"), 1)
		m.Set([]byte("b"), 2)
		m.Set([]byte("c"), 3)

		Expect(t, m, map[string]int{
			"b": 2,
			"c": 3,
			"a": 1,
		})

		m.Delete([]byte("a"))
		Expect(t, m, map[string]int{
			"b": 2,
			"c": 3,
		})

		m.Delete([]byte("b"))
		Expect(t, m, map[string]int{
			"c": 3,
		})

		m.Delete([]byte("c"))
		Expect(t, m, nil)

		m.Delete([]byte("a"))
		m.Delete([]byte("b"))
		m.Delete([]byte("c"))
		Expect(t, m, nil)
	})
}

func TestLen(t *testing.T) {
	forEachImplT(t, func(t *testing.T, m container.Mapper[[]byte, int]) {
		dataSet := make([]string, 512)
		for i := range dataSet {
			dataSet[i] = strconv.Itoa(i)
		}
		for i, d := range dataSet {
			m.Set([]byte(d), i)
		}
		require.Equal(t, len(dataSet), m.Len())
	})
}

func Expect[K container.KeyInterface, V any](
	t *testing.T,
	a container.Mapper[K, V],
	expect map[string]V,
) {
	t.Helper()
	require.Equal(t, len(expect), a.Len())
	for k, ev := range expect {
		v, ok := a.Get(K(k))
		require.True(t, ok)
		require.Equal(t, ev, v)
	}
}

func HasVal[K container.KeyInterface, V any](
	t *testing.T,
	m container.Mapper[K, V],
	key K,
	expectedValue V,
) {
	t.Helper()
	v, ok := m.Get(key)
	require.True(t, ok)
	require.Equal(t, expectedValue, v)
}
