package stack_test

import (
	"testing"

	"github.com/graph-guard/ggproxy/utilities/stack"
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
	require.Equal(t, int64(-1), st.Pop())
	st.Pop()
	st.Pop()
	require.Equal(t, int64(0), st.Pop())
}

func TestPopPush(t *testing.T) {
	st := stack.New[float64](2)
	st.Push(0.0)
	require.Equal(t, 0.0, st.PopPush(-1))
	require.Equal(t, -1.0, st.Pop())
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
	require.Equal(t, 20, st.Pop())
	require.Equal(t, 10, st.Pop())
	require.Equal(t, 0, st.Top())
}

func TestGet(t *testing.T) {
	st := stack.New[int](2)
	st.Push(0)
	st.Push(-1)
	require.Equal(t, 0, st.Get(0))
	require.Equal(t, -1, st.Get(1))
}
