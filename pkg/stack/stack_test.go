package stack_test

import (
	"testing"

	"github.com/graph-guard/ggproxy/pkg/stack"
	"github.com/stretchr/testify/require"
)

func TestReset(t *testing.T) {
	st := stack.New[uint16](2)
	st.Push(0)
	st.Reset()
	require.Equal(t, stack.New[uint16](2), st)
}

func TestPushLen(t *testing.T) {
	st := stack.New[int16](4)
	st.Push(0)
	st.Push(1)
	st.Push(-1)
	require.Equal(t, 3, st.Len())
}

func TestPop(t *testing.T) {
	st := stack.New[int64](4)
	st.Push(0)
	st.Push(1)
	st.Push(-1)
	require.Equal(t, int64(-1), st.TopPop())
	st.Pop()
	st.Pop()
	require.Equal(t, int64(0), st.TopPop())
}

func TestTopPopPush(t *testing.T) {
	st := stack.New[float64](2)
	st.Push(0.0)
	require.Equal(t, 0.0, st.TopPopPush(-1))
	require.Equal(t, -1.0, st.TopPop())
}

func TestPopPush(t *testing.T) {
	st := stack.New[float64](2)
	st.Push(0.0)
	st.PopPush(-1)
	require.Equal(t, -1.0, st.Top())
	st.PopPush(3.14)
	require.Equal(t, 3.14, st.Top())
}

func TestTop(t *testing.T) {
	st := stack.New[int](2)
	st.Push(0)
	st.Push(-1)
	require.Equal(t, -1, st.Top())
	st.Pop()
	st.Pop()
	require.Equal(t, 0, st.Top())
}

func TestTopOffset(t *testing.T) {
	st := stack.New[int](2)
	st.Push(1)
	st.Push(2)
	st.Push(3)
	require.Equal(t, 3, st.TopOffset(0))
	require.Equal(t, 2, st.TopOffset(1))
	require.Equal(t, 1, st.TopOffset(2))
	require.Zero(t, st.TopOffset(3))
	st.Pop()
	require.Equal(t, 2, st.TopOffset(0))
	require.Equal(t, 1, st.TopOffset(1))
	require.Zero(t, st.TopOffset(2))
	require.Zero(t, st.TopOffset(3))
}

func TestTopOffsetNeg(t *testing.T) {
	st := stack.New[int](2)
	st.Push(1)
	st.Push(2)
	require.Panics(t, func() {
		require.Zero(t, st.TopOffset(-1))
	})
	require.Panics(t, func() {
		require.Zero(t, st.TopOffset(-2))
	})
}

func TestTopOffsetFn(t *testing.T) {
	st := stack.New[int](2)
	st.Push(0)
	st.Push(-1)
	st.TopOffsetFn(0, func(i *int) {
		require.Equal(t, -1, *i)
		*i = 20
	})
	st.TopOffsetFn(1, func(i *int) {
		require.Equal(t, 0, *i)
		*i = 10
	})
	require.Equal(t, 20, st.TopPop())
	st.Pop() // 10
	require.Equal(t, 0, st.Top())
}

func TestTopOffsetFnNeg(t *testing.T) {
	st := stack.New[int](2)
	st.Push(1)
	st.Push(2)
	require.Panics(t, func() {
		st.TopOffsetFn(-1, func(t *int) {})
	})
	require.Panics(t, func() {
		st.TopOffsetFn(-2, func(t *int) {})
	})
}

func TestGet(t *testing.T) {
	st := stack.New[int](2)
	st.Push(0)
	st.Push(-1)
	require.Equal(t, 0, st.Get(0))
	require.Equal(t, -1, st.Get(1))
}
