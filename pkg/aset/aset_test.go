package aset_test

import (
	"testing"

	"github.com/graph-guard/ggproxy/pkg/aset"
	"github.com/graph-guard/ggproxy/pkg/math"
	"github.com/stretchr/testify/require"
)

func TestReset(t *testing.T) {
	s := aset.New[uint16](8, 0, 1, 2, 3)
	s.Reset()
	require.Equal(t, aset.New[uint16](8), s)
}

func TestAdd(t *testing.T) {
	s := aset.New[int16](8)
	s.Add(0)
	s.Add(1)
	s.Add(-1)
	Expect(t, s, -1, 0, 1)
}

func TestGet(t *testing.T) {
	s := aset.New[float64](8, 2, -1)
	require.Equal(t, -1.0, s.Get(0))
}

func TestDelete(t *testing.T) {
	s := aset.New[int64](8, 2, -1, 27015)
	require.Equal(t, int64(27015), s.Delete(2))
	Expect(t, s, -1, 2)
}

func TestFind(t *testing.T) {
	dataSet := make([]uint64, 512)
	for i := range dataSet {
		dataSet[i] = RandUint64()
	}
	s := aset.New(1024, dataSet...)
	s.Add(69)
	require.NotEqual(t, -1, s.Find(69))
}

func TestFoundNothing(t *testing.T) {
	s := aset.New[int32](8, 2, -1, 27015, 42, -273)
	require.Equal(t, -1, s.Find(69))
}

func TestLen(t *testing.T) {
	dataSet := make([]uint64, 512)
	for i := range dataSet {
		dataSet[i] = RandUint64()
	}
	s := aset.New(1024, dataSet...)
	require.Equal(t, 512, s.Len())
}

func TestVisit(t *testing.T) {
	s := aset.New[uint64](8, 27015)
	s.Visit(func(e uint64) (stop bool) {
		require.Equal(t, uint64(27015), e)
		return false
	})
}

func TestVisitStop(t *testing.T) {
	s := aset.New[uint64](8, 27015)
	s.Visit(func(e uint64) (stop bool) {
		require.Equal(t, uint64(27015), e)
		return true
	})
}

func Expect[T math.NumberInterface](t *testing.T, a *aset.Set[T], e ...T) {
	t.Helper()
	var elements []T
	require.Equal(t, len(e), a.Len())
	a.Visit(func(t T) (stop bool) {
		elements = append(elements, t)
		return false
	})
	require.Equal(t, e, elements)
}
