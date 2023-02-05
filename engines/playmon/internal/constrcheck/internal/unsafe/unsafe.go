package unsafe

import "unsafe"

// B2S returns a string representation of b.
func B2S(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}
