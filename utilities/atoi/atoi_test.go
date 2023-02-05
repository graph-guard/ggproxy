package atoi_test

import (
	"fmt"
	"math"
	"strconv"
	"testing"

	"github.com/graph-guard/ggproxy/utilities/atoi"
	"github.com/stretchr/testify/require"
)

// Std wraps strconv.Atoi.
func Std[S []byte | string](s S) int32 {
	i, _ := strconv.Atoi(string(s))
	return int32(i)
}

func TestMustI32(t *testing.T) {
	require.Equal(t, int32(0), atoi.MustI32("0"))
	require.Equal(t, int32(1), atoi.MustI32("1"))
	require.Equal(t, int32(8), atoi.MustI32("8"))
	require.Equal(t, int32(-1), atoi.MustI32("-1"))
	require.Equal(t, int32(1), atoi.MustI32("+1"))
	require.Equal(t, int32(123456789), atoi.MustI32("123456789"))
	require.Equal(t, int32(1234567890), atoi.MustI32("1234567890"))
	require.Equal(t, int32(math.MaxInt32), atoi.MustI32(fmt.Sprintf("%d", math.MaxInt32)))
	require.Equal(t, int32(math.MinInt32), atoi.MustI32(fmt.Sprintf("%d", math.MinInt32)))

	// Error
	require.Panics(t, func() { atoi.MustI32("a") })
	require.Panics(t, func() { atoi.MustI32("0xa") })
	require.Panics(t, func() { atoi.MustI32(" 1") })
	require.Panics(t, func() { atoi.MustI32("-0xa") })
	require.Panics(t, func() { atoi.MustI32("-") })
	require.Panics(t, func() { atoi.MustI32("") })
}

func TestMustF64(t *testing.T) {
	require.Equal(t, float64(0), atoi.MustF64("0"))
	require.Equal(t, float64(1), atoi.MustF64("1"))
	require.Equal(t, float64(3.14), atoi.MustF64("3.14"))
	require.Equal(t, float64(-1), atoi.MustF64("-1"))
	require.Equal(t, float64(-1.12345), atoi.MustF64("-1.12345"))
	require.Equal(t, float64(1), atoi.MustF64("+1"))
	require.Equal(t, float64(123456789), atoi.MustF64("123456789"))
	require.Equal(t, float64(1234567890), atoi.MustF64("1234567890"))
	require.Equal(t, float64(0.1234567890), atoi.MustF64("0.1234567890"))
	require.Equal(t,
		float64(math.MaxFloat64),
		atoi.MustF64(fmt.Sprintf("%f", math.MaxFloat64)),
	)
	require.Equal(t,
		float64(-math.MaxFloat64),
		atoi.MustF64(fmt.Sprintf("%f", -math.MaxFloat64)),
	)
	require.Equal(t, float64(1e12), atoi.MustF64("1e12"))
	require.Equal(t, float64(1e+12), atoi.MustF64("1e+12"))
	require.Equal(t, float64(1e-12), atoi.MustF64("1e-12"))
	require.Equal(t, float64(1e-200), atoi.MustF64("1e-200"))
	require.Equal(t, float64(1e308), atoi.MustF64("1e308"))
	require.Equal(t, float64(1e-308), atoi.MustF64("1e-308"))

	// Error
	require.Panics(t, func() { atoi.MustF64("a") })
	require.Panics(t, func() { atoi.MustF64("0xa") })
	require.Panics(t, func() { atoi.MustF64(" 1") })
	require.Panics(t, func() { atoi.MustF64("-0xa") })
	require.Panics(t, func() { atoi.MustF64("-") })
	require.Panics(t, func() { atoi.MustF64("") })
	require.Panics(t, func() { atoi.MustF64(".") })
	require.Panics(t, func() { atoi.MustF64("1.2.3") })
}

var GI32 int32

func BenchmarkI32(b *testing.B) {
	for _, bb := range []struct {
		Name  string
		Input string
	}{
		{"min", fmt.Sprintf("%d", math.MinInt32)},
		{"1", fmt.Sprintf("%d", 1)},
		{"123456789", fmt.Sprintf("%d", 123456789)},
		{"max", fmt.Sprintf("%d", math.MaxInt32)},
		{"plus_prefix", "+1"},
	} {
		b.Run(bb.Name, func(b *testing.B) {
			b.Run("string", func(b *testing.B) {
				b.Run("std", func(b *testing.B) {
					for i := 0; i < b.N; i++ {
						GI32 = Std(bb.Input)
					}
				})
				b.Run("custom", func(b *testing.B) {
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						GI32 = atoi.MustI32(bb.Input)
					}
				})
			})
			b.Run("byte_slice", func(b *testing.B) {
				s := []byte(bb.Input)
				b.Run("std", func(b *testing.B) {
					for i := 0; i < b.N; i++ {
						GI32 = Std(s)
					}
				})
				b.Run("custom", func(b *testing.B) {
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						GI32 = atoi.MustI32(s)
					}
				})
			})
		})
	}
}
