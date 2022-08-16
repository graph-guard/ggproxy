package amap_test

import (
	"testing"

	"github.com/graph-guard/ggproxy/utilities/container/amap"
	"github.com/stretchr/testify/require"
)

func TestReset(t *testing.T) {
	m := amap.New[int, bool](8)

	numKeys := 5
	for i := 0; i < numKeys; i++ {
		m.Set(i, true)
	}
	require.Equal(t, numKeys, m.Len())

	m.Reset()

	require.Zero(t, m.Len())
	for i := 0; i < numKeys; i++ {
		require.Equal(t, -1, m.Index(i))
	}
}

func TestSet(t *testing.T) {
	m := amap.New[int, int16](8)
	m.Set(1, -1)
	m.Set(2, 1)
	m.Set(0, 0)
	Expect(t, m,
		[]int{0, 1, 2},
		[]int16{0, -1, 1},
	)

	m.Set(0, 2)
	m.Set(1, 3)
	m.Set(2, 4)
	Expect(t, m,
		[]int{0, 1, 2},
		[]int16{2, 3, 4},
	)
}

func TestSetFn(t *testing.T) {
	m := amap.New(8,
		amap.Entity[int, int16]{Key: 1, Value: -1},
		amap.Entity[int, int16]{Key: 2, Value: 1},
		amap.Entity[int, int16]{Key: 0, Value: 0},
	)
	Expect(t, m,
		[]int{0, 1, 2},
		[]int16{0, -1, 1},
	)

	m.SetFn(0, 1, func(value *int16) { *value++ })
	m.SetFn(1, 3, func(value *int16) { *value++ })
	m.SetFn(2, 4, func(value *int16) { *value-- })
	Expect(t, m,
		[]int{0, 1, 2},
		[]int16{1, 0, 0},
	)
}

func TestGet(t *testing.T) {
	m := amap.New(8,
		amap.Entity[int, float32]{Key: -1, Value: 2},
		amap.Entity[int, float32]{Key: 1, Value: 3},
	)

	{
		v, ok := m.Get(-1)
		require.True(t, ok)
		require.Equal(t, float32(2), v)
	}
	{
		v, ok := m.Get(-2)
		require.False(t, ok)
		require.Zero(t, v)
	}
}

func TestDelete(t *testing.T) {
	m := amap.New(8,
		amap.Entity[int, int]{Key: 0, Value: 1},
		amap.Entity[int, int]{Key: 1, Value: 2},
		amap.Entity[int, int]{Key: 2, Value: 3},
	)

	Expect(t, m,
		[]int{0, 1, 2},
		[]int{1, 2, 3},
	)

	k, v := m.DeleteByIndex(10)
	require.Zero(t, k)
	require.Zero(t, v)

	k, v = m.DeleteByIndex(m.Index(1))
	require.Equal(t, 1, k)
	require.Equal(t, 2, v)

	Expect(t, m,
		[]int{0, 2},
		[]int{1, 3},
	)

	k, v = m.DeleteByIndex(m.Index(2))
	require.Equal(t, 2, k)
	require.Equal(t, 3, v)

	Expect(t, m,
		[]int{0},
		[]int{1},
	)

	k, v = m.DeleteByIndex(m.Index(0))
	require.Equal(t, 0, k)
	require.Equal(t, 1, v)

	Expect(t, m, []int(nil), []int(nil))

	k, v = m.DeleteByIndex(10)
	require.Zero(t, k)
	require.Zero(t, v)
}

func TestFind(t *testing.T) {
	dataSet := make([]int, 512)
	for i := range dataSet {
		dataSet[i] = i
	}
	m := amap.New[int, bool](1024)
	for _, d := range dataSet {
		require.Equal(t, -1, m.Index(d))
	}
	for _, el := range dataSet {
		m.Set(el, true)
	}
	for i := 512; i < 1024; i++ {
		require.Equal(t, -1, m.Index(i))
	}
}

func TestLen(t *testing.T) {
	dataSet := make([]int, 512)
	for i := range dataSet {
		dataSet[i] = i
	}
	m := amap.New[int, bool](1024)
	for _, el := range dataSet {
		m.Set(el, true)
	}
	require.Equal(t, 512, m.Len())
}

func TestVisitStop(t *testing.T) {
	m := amap.New(8,
		amap.Entity[int, string]{Key: 0, Value: "val1"},
		amap.Entity[int, string]{Key: 1, Value: "val2"},
		amap.Entity[int, string]{Key: 2, Value: "val3"},
	)
	calls := 0
	m.Visit(func(k int, v string) (stop bool) {
		require.Equal(t, 0, k)
		require.Equal(t, "val1", v)
		calls++
		return true
	})
	require.Equal(t, 1, calls)
}

func Expect[K amap.KeyInterface, V any](
	t *testing.T,
	a *amap.Map[K, V],
	keys []K,
	values []V,
) {
	t.Helper()
	var actualKeys []K
	var actualValues []V
	require.Equal(t, len(keys), a.Len())
	a.Visit(func(key K, value V) (stop bool) {
		actualKeys = append(actualKeys, key)
		actualValues = append(actualValues, value)
		return false
	})
	require.Equal(t, keys, actualKeys)
	require.Equal(t, values, actualValues)
}
