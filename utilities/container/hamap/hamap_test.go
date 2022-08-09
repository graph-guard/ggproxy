package hamap_test

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/graph-guard/gguard-proxy/utilities/container/hamap"
	"github.com/stretchr/testify/require"
)

func TestReset(t *testing.T) {
	m := hamap.New[string, bool](8, &MockHasher[string]{
		Map: map[string]uint64{"0": 0, "1": 1, "2": 2, "3": 3, "4": 4},
	})

	numKeys := 5
	for i := 0; i < numKeys; i++ {
		m.Set(strconv.Itoa(i), true)
	}
	require.Equal(t, numKeys, m.Len())

	m.Reset()

	require.Zero(t, m.Len())
	for i := 0; i < numKeys; i++ {
		v, ok := m.Get(strconv.Itoa(i))
		require.False(t, ok)
		require.Zero(t, v)
	}
}

func TestDefaultHasher(t *testing.T) {
	t.Run("bytes", func(t *testing.T) {
		m := hamap.New[[]byte, int](8, nil)
		m.Set([]byte("key"), 1)
	})
	t.Run("string", func(t *testing.T) {
		m := hamap.New[string, int](8, nil)
		m.Set("key", 1)
	})
}

func TestSet(t *testing.T) {
	m := hamap.New[[]byte, int](8, &MockHasher[[]byte]{
		Map: map[string]uint64{"x": 0, "a": 1, "b": 2, "c": 3},
	})
	m.Set([]byte("a"), -1)
	m.Set([]byte("b"), 0)
	m.Set([]byte("c"), 1)
	Expect(t, m,
		[][]byte{[]byte("a"), []byte("b"), []byte("c")},
		[]int{-1, 0, 1},
	)

	m.Set([]byte("a"), 2)
	m.Set([]byte("b"), 3)
	m.Set([]byte("c"), 4)
	Expect(t, m,
		[][]byte{[]byte("a"), []byte("b"), []byte("c")},
		[]int{2, 3, 4},
	)

	m.Set([]byte("x"), 42)
	Expect(t, m,
		[][]byte{[]byte("x"), []byte("a"), []byte("b"), []byte("c")},
		[]int{42, 2, 3, 4},
	)
}

func TestSetCollision(t *testing.T) {
	m := hamap.New[[]byte, int](8, &MockHasher[[]byte]{
		Map: map[string]uint64{"x": 0, "a": 1, "b": 2, "c": 2, "d": 2},
	})
	m.Set([]byte("a"), -1)
	m.Set([]byte("b"), 0)
	m.Set([]byte("c"), 1)
	Expect(t, m,
		[][]byte{[]byte("a"), []byte("b"), []byte("c")},
		[]int{-1, 0, 1},
	)

	m.Set([]byte("a"), 2)
	m.Set([]byte("b"), 3)
	m.Set([]byte("c"), 4)
	m.Set([]byte("d"), 11)
	Expect(t, m,
		[][]byte{[]byte("a"), []byte("b"), []byte("c"), []byte("d")},
		[]int{2, 3, 4, 11},
	)

	m.Set([]byte("x"), 42)
	Expect(t, m,
		[][]byte{
			[]byte("x"),
			[]byte("a"),
			[]byte("b"),
			[]byte("c"),
			[]byte("d"),
		},
		[]int{42, 2, 3, 4, 11},
	)
}

func TestSetFn(t *testing.T) {
	m := hamap.New[[]byte, int](8, &MockHasher[[]byte]{
		Map: map[string]uint64{"x": 0, "a": 1, "b": 2, "c": 3},
	})
	m.SetFn([]byte("a"), func(v *int) int {
		require.Nil(t, v)
		return -1
	})
	m.SetFn([]byte("b"), func(v *int) int {
		require.Nil(t, v)
		return 0
	})
	m.SetFn([]byte("c"), func(v *int) int {
		require.Nil(t, v)
		return 1
	})
	Expect(t, m,
		[][]byte{[]byte("a"), []byte("b"), []byte("c")},
		[]int{-1, 0, 1},
	)

	m.SetFn([]byte("a"), func(v *int) int {
		require.NotNil(t, v)
		*v = 2
		return 0
	})
	m.SetFn([]byte("b"), func(v *int) int {
		require.NotNil(t, v)
		*v = 3
		return 0
	})
	m.SetFn([]byte("c"), func(v *int) int {
		require.NotNil(t, v)
		*v = 4
		return 0
	})
	Expect(t, m,
		[][]byte{[]byte("a"), []byte("b"), []byte("c")},
		[]int{2, 3, 4},
	)

	m.SetFn([]byte("x"), func(v *int) int {
		require.Nil(t, v)
		return 42
	})
	Expect(t, m,
		[][]byte{[]byte("x"), []byte("a"), []byte("b"), []byte("c")},
		[]int{42, 2, 3, 4},
	)
}

func TestSetFnCollision(t *testing.T) {
	m := hamap.New[[]byte, int](8, &MockHasher[[]byte]{
		Map: map[string]uint64{"x": 0, "a": 1, "b": 2, "c": 2, "d": 2},
	})
	m.SetFn([]byte("a"), func(v *int) int {
		require.Nil(t, v)
		return -1
	})
	m.SetFn([]byte("b"), func(v *int) int {
		require.Nil(t, v)
		return 0
	})
	m.SetFn([]byte("c"), func(v *int) int {
		require.Nil(t, v)
		return 1
	})
	Expect(t, m,
		[][]byte{[]byte("a"), []byte("b"), []byte("c")},
		[]int{-1, 0, 1},
	)

	m.SetFn([]byte("a"), func(v *int) int {
		require.NotNil(t, v)
		*v = 2
		return 0
	})
	m.SetFn([]byte("b"), func(v *int) int {
		require.NotNil(t, v)
		*v = 3
		return 0
	})
	m.SetFn([]byte("c"), func(v *int) int {
		require.NotNil(t, v)
		*v = 4
		return 0
	})
	m.SetFn([]byte("d"), func(v *int) int {
		require.Nil(t, v)
		return 11
	})
	Expect(t, m,
		[][]byte{[]byte("a"), []byte("b"), []byte("c"), []byte("d")},
		[]int{2, 3, 4, 11},
	)

	m.SetFn([]byte("x"), func(v *int) int {
		require.Nil(t, v)
		return 42
	})
	Expect(t, m,
		[][]byte{
			[]byte("x"),
			[]byte("a"),
			[]byte("b"),
			[]byte("c"),
			[]byte("d"),
		},
		[]int{42, 2, 3, 4, 11},
	)
}

func TestGet(t *testing.T) {
	m := hamap.New[string, float32](8, &MockHasher[string]{
		Map: map[string]uint64{"a": 0, "b": 1, "nonexistent": 2},
	})
	m.Set("a", 2)
	m.Set("b", 3)

	{
		v, ok := m.Get("b")
		require.True(t, ok)
		require.Equal(t, float32(3), v)
	}
	{
		v, ok := m.Get("nonexistent")
		require.False(t, ok)
		require.Zero(t, v)
	}
}

func TestGetCollision(t *testing.T) {
	m := hamap.New[[]byte, int](8, &MockHasher[[]byte]{
		Map: map[string]uint64{"x": 0, "a": 1, "b": 2, "c": 2, "d": 2},
	})
	m.Set([]byte("a"), 2)
	m.Set([]byte("b"), 3)
	m.Set([]byte("c"), 4)

	{
		v, ok := m.Get([]byte("a"))
		require.True(t, ok)
		require.Equal(t, 2, v)
	}
	{
		v, ok := m.Get([]byte("b"))
		require.True(t, ok)
		require.Equal(t, 3, v)
	}
	{
		v, ok := m.Get([]byte("c"))
		require.True(t, ok)
		require.Equal(t, 4, v)
	}
	{
		v, ok := m.Get([]byte("d"))
		require.False(t, ok)
		require.Zero(t, v)
	}
	{
		v, ok := m.Get([]byte("x"))
		require.False(t, ok)
		require.Zero(t, v)
	}
}

func TestGetFn(t *testing.T) {
	m := hamap.New[string, float32](8, &MockHasher[string]{
		Map: map[string]uint64{"a": 0, "b": 1, "nonexistent": 2},
	})
	m.Set("a", 2)
	m.Set("b", 3)

	{
		ok := m.GetFn("b", func(v *float32) {
			require.NotNil(t, v)
			require.Equal(t, float32(3), *v)
			*v = 42.5 // Mutate value!
		})
		require.True(t, ok)
	}
	{
		ok := m.GetFn("b", func(v *float32) {
			require.NotNil(t, v)
			require.Equal(t, float32(42.5), *v)
		})
		require.True(t, ok)
	}
	{
		ok := m.GetFn("nonexistent", func(v *float32) {
			t.Fatal("this function is expected not to be called!")
		})
		require.False(t, ok)
	}
}

func TestGetFnCollision(t *testing.T) {
	m := hamap.New[[]byte, int](8, &MockHasher[[]byte]{
		Map: map[string]uint64{"x": 0, "a": 1, "b": 2, "c": 2, "d": 2},
	})
	m.Set([]byte("a"), 2)
	m.Set([]byte("b"), 3)
	m.Set([]byte("c"), 4)

	{
		ok := m.GetFn([]byte("a"), func(v *int) {
			require.NotNil(t, v)
			require.Equal(t, 2, *v)
		})
		require.True(t, ok)
	}
	{
		ok := m.GetFn([]byte("b"), func(v *int) {
			require.NotNil(t, v)
			require.Equal(t, 3, *v)
		})
		require.True(t, ok)
	}
	{
		ok := m.GetFn([]byte("c"), func(v *int) {
			require.NotNil(t, v)
			require.Equal(t, 4, *v)
		})
		require.True(t, ok)
	}
	{
		ok := m.GetFn([]byte("d"), func(v *int) {
			t.Fatal("this function is expected not to be called!")
		})
		require.False(t, ok)
	}
	{
		ok := m.GetFn([]byte("x"), func(v *int) {
			t.Fatal("this function is expected not to be called!")
		})
		require.False(t, ok)
	}
}

func TestDelete(t *testing.T) {
	m := hamap.New[[]byte, int](8, &MockHasher[[]byte]{
		Map: map[string]uint64{"a": 0, "b": 1, "c": 2, "d": 3},
	})
	m.Set([]byte("a"), 1)
	m.Set([]byte("b"), 2)
	m.Set([]byte("c"), 3)
	Expect(t, m,
		[][]byte{[]byte("a"), []byte("b"), []byte("c")},
		[]int{1, 2, 3},
	)

	m.Delete([]byte("a"))
	Expect(t, m,
		[][]byte{[]byte("b"), []byte("c")},
		[]int{2, 3},
	)

	m.Delete([]byte("b"))
	Expect(t, m,
		[][]byte{[]byte("c")},
		[]int{3},
	)

	m.Delete([]byte("c"))
	Expect(t, m, [][]byte(nil), []int(nil))

	m.Delete([]byte("d"))
	Expect(t, m, [][]byte(nil), []int(nil))
}

func TestDeleteCollision(t *testing.T) {
	m := hamap.New[[]byte, int](8, &MockHasher[[]byte]{
		Map: map[string]uint64{
			"a": 0, "b": 1, "c": 1, "d": 3,
			"col2_1": 4, "col2_2": 4,
			"col3_1": 5, "col3_2": 5, "col3_3": 5, "col3_4": 5,
		},
	})
	m.Set([]byte("a"), 1)
	m.Set([]byte("b"), 2)
	m.Set([]byte("c"), 3)
	m.Set([]byte("col2_1"), 4)
	m.Set([]byte("col2_2"), 5)
	m.Set([]byte("col3_1"), 6)
	m.Set([]byte("col3_2"), 7)
	m.Set([]byte("col3_3"), 8)
	Expect(t, m,
		[][]byte{
			[]byte("a"), []byte("b"), []byte("c"),
			[]byte("col2_1"), []byte("col2_2"),
			[]byte("col3_1"), []byte("col3_2"), []byte("col3_3"),
		},
		[]int{1, 2, 3, 4, 5, 6, 7, 8},
	)
	require.Equal(t, 8, m.Len())

	m.Delete([]byte("a"))
	Expect(t, m,
		[][]byte{
			[]byte("b"), []byte("c"),
			[]byte("col2_1"), []byte("col2_2"),
			[]byte("col3_1"), []byte("col3_2"), []byte("col3_3"),
		},
		[]int{2, 3, 4, 5, 6, 7, 8},
	)
	require.Equal(t, 7, m.Len())

	m.Delete([]byte("b"))
	Expect(t, m,
		[][]byte{
			[]byte("c"),
			[]byte("col2_1"), []byte("col2_2"),
			[]byte("col3_1"), []byte("col3_2"), []byte("col3_3"),
		},
		[]int{3, 4, 5, 6, 7, 8},
	)
	require.Equal(t, 6, m.Len())

	m.Delete([]byte("c"))
	Expect(t, m,
		[][]byte{
			[]byte("col2_1"), []byte("col2_2"),
			[]byte("col3_1"), []byte("col3_2"), []byte("col3_3"),
		},
		[]int{4, 5, 6, 7, 8},
	)
	require.Equal(t, 5, m.Len())

	m.Delete([]byte("col2_2"))
	Expect(t, m,
		[][]byte{
			[]byte("col2_1"),
			[]byte("col3_1"), []byte("col3_2"), []byte("col3_3"),
		},
		[]int{4, 6, 7, 8},
	)
	require.Equal(t, 4, m.Len())

	m.Delete([]byte("col2_1"))
	Expect(t, m,
		[][]byte{
			[]byte("col3_1"), []byte("col3_2"), []byte("col3_3"),
		},
		[]int{6, 7, 8},
	)
	require.Equal(t, 3, m.Len())

	m.Delete([]byte("col3_2"))
	Expect(t, m,
		[][]byte{
			[]byte("col3_1"), []byte("col3_3"),
		},
		[]int{6, 8},
	)
	require.Equal(t, 2, m.Len())

	m.Delete([]byte("col3_3"))
	Expect(t, m,
		[][]byte{
			[]byte("col3_1"),
		},
		[]int{6},
	)
	require.Equal(t, 1, m.Len())

	m.Delete([]byte("d"))
	Expect(t, m,
		[][]byte{
			[]byte("col3_1"),
		},
		[]int{6},
	)
	require.Equal(t, 1, m.Len())

	m.Delete([]byte("col3_4"))
	require.Equal(t, 1, m.Len())
	Expect(t, m,
		[][]byte{
			[]byte("col3_1"),
		},
		[]int{6},
	)
}

func TestLen(t *testing.T) {
	keys := make([]string, 5)
	for i := range keys {
		keys[i] = strconv.Itoa(i)
	}
	m := hamap.New[string, bool](1024, &MockHasher[string]{
		Map: map[string]uint64{"0": 0, "1": 1, "2": 2, "3": 3, "4": 4},
	})
	for i, el := range keys {
		m.Set(el, true)
		require.Equal(t, i+1, m.Len())
	}
	require.Equal(t, len(keys), m.Len())
}

func TestVisitStop(t *testing.T) {
	m := hamap.New[string, string](8, &MockHasher[string]{
		Map: map[string]uint64{"a": 0, "b": 1, "c": 2, "d": 2},
	})
	m.Set("a", "val1")
	m.Set("b", "val2")
	m.Set("c", "val3")
	m.Set("d", "val4")
	calls := 0
	m.Visit(func(k string, v string) (stop bool) {
		require.Equal(t, "a", k)
		require.Equal(t, "val1", v)
		calls++
		return true
	})
	require.Equal(t, 1, calls)
	calls = 0
	m.Visit(func(k string, v string) (stop bool) {
		calls++
		return calls == 4
	})
	require.Equal(t, 4, calls)
}

func TestGet512(t *testing.T) {
	m := hamap.New[string, int](8, nil)
	for i := 0; i < 512; i++ {
		m.Set(strconv.Itoa(i), i)
	}
	for i := 0; i < 512; i++ {
		v, ok := m.Get(strconv.Itoa(i))
		require.True(t, ok)
		require.Equal(t, i, v)
	}
}

func Expect[K hamap.KeyInterface, V any](
	t *testing.T,
	a *hamap.Map[K, V],
	keys []K,
	values []V,
) {
	t.Helper()
	var actualKeys []K
	var actualValues []V
	require.Equal(t, len(keys), a.Len())
	a.VisitAll(func(key K, value V) {
		actualKeys = append(actualKeys, key)
		actualValues = append(actualValues, value)
	})
	require.Equal(t, keys, actualKeys)
	require.Equal(t, values, actualValues)
}

type MockHasher[K hamap.KeyInterface] struct {
	Map map[string]uint64
}

func (m *MockHasher[K]) Hash(k K) uint64 {
	if hashValue, ok := m.Map[string(k)]; ok {
		return hashValue
	}
	panic(fmt.Errorf("missing hash value for key %q", string(k)))
}
