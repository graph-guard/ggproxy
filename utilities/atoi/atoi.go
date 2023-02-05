// Package atoi provides functions for efficient & allocation-free
// parsing of []byte and string to int32 and float64.
package atoi

import (
	"fmt"
	"strconv"

	"github.com/graph-guard/ggproxy/utilities/unsafe"
)

// MustI32 parses s assuming that it's a valid signed 32-bit integer.
// Panics if s contains an invalid number.
func MustI32[S []byte | string](s S) int32 {
	const intSize = 32 << (^uint(0) >> 63)

	sLen := len(s)
	if intSize == 32 && (0 < sLen && sLen < 10) ||
		intSize == 64 && (0 < sLen && sLen < 19) {
		// Fast path for small integers that fit int type.
		s0 := s
		if s[0] == '-' || s[0] == '+' {
			s = s[1:]
			if len(s) < 1 {
				panic("syntax error")
			}
		}

		n := int32(0)
		for _, ch := range []byte(s) {
			ch -= '0'
			if ch > 9 {
				panic("syntax error")
			}
			n = n*10 + int32(ch)
		}
		if s0[0] == '-' {
			n = -n
		}
		return n
	}
	panic("syntax error")
}

// MustF64 parses s assuming that it's a valid signed 64-bit float.
// Panics if s contains an invalid number.
func MustF64[S []byte | string](s S) float64 {
	f, err := strconv.ParseFloat(unsafe.B2S([]byte(s)), 64)
	if err != nil {
		panic(fmt.Errorf("unexpected float64 parsing err: %w", err))
	}
	return f
}
