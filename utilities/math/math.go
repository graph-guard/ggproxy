package math

// NumberInterface is a generic number interface for all number types.
type NumberInterface interface {
	uint8 | uint16 | uint32 | uint64 | int | int8 | int16 | int32 | int64 | float32 | float64
}

// Max calculates the maximum of two numbers.
func Max[T NumberInterface](a, b T) T {
	if a > b {
		return a
	}
	return b
}

// Min calculates the minimum of two numbers.
func Min[T NumberInterface](a, b T) T {
	if a < b {
		return a
	}
	return b
}
