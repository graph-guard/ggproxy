package math_test

import (
	"testing"

	"github.com/graph-guard/ggproxy/pkg/math"
	"github.com/stretchr/testify/require"
)

func TestMax(t *testing.T) {
	require.Equal(t, 1.0, math.Max(-1.0, 1.0))
	require.Equal(t, 1.0, math.Max(1.0, -1.0))
}

func TestMin(t *testing.T) {
	require.Equal(t, -1.0, math.Min(-1.0, 1.0))
	require.Equal(t, -1.0, math.Min(1.0, -1.0))
}
