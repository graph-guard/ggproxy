// Package xxhash provides XXH64 hashing capabilities.
//
// Forked from github.com/pierrec/xxHash.
package xxhash

const (
	prime64_1 = 11400714785074694791
	prime64_2 = 14029467366897019727
	prime64_3 = 1609587929392839161
	prime64_4 = 9650029242287828579
	prime64_5 = 2870177450012600261
)

// Hash is a hash object.
type Hash struct {
	seed, v1, v2, v3, v4, totalLen uint64
	buf                            [32]byte
	bufused                        int
}

// New returns a new Hash64 instance.
func New(seed uint64) Hash {
	return Hash{
		seed: seed,
		v1:   seed + prime64_1 + prime64_2,
		v2:   seed + prime64_2,
		v3:   seed,
		v4:   seed - prime64_1,
	}
}

// Write adds input bytes to the output hash without mutating the hash.
func Write[B []byte | string](h *Hash, input B) {
	n := len(input)
	m := h.bufused

	h.totalLen += uint64(n)

	r := len(h.buf) - m
	if n < r {
		copy(h.buf[m:], input)
		h.bufused += len(input)
		return
	}

	p := 0
	if m > 0 {
		// some data left from previous update
		copy(h.buf[h.bufused:], input[:r])
		h.bufused += len(input) - r

		// fast rotl(31)
		h.v1 = rol31(h.v1+u64(h.buf[:])*prime64_2) * prime64_1
		h.v2 = rol31(h.v2+u64(h.buf[8:])*prime64_2) * prime64_1
		h.v3 = rol31(h.v3+u64(h.buf[16:])*prime64_2) * prime64_1
		h.v4 = rol31(h.v4+u64(h.buf[24:])*prime64_2) * prime64_1
		p = r
		h.bufused = 0
	}

	// Causes compiler to work directly from registers instead of stack:
	v1, v2, v3, v4 := h.v1, h.v2, h.v3, h.v4
	for n := n - 32; p <= n; p += 32 {
		sub := input[p:][:32] // BCE hint for compiler
		v1 = rol31(v1+u64(sub[:])*prime64_2) * prime64_1
		v2 = rol31(v2+u64(sub[8:])*prime64_2) * prime64_1
		v3 = rol31(v3+u64(sub[16:])*prime64_2) * prime64_1
		v4 = rol31(v4+u64(sub[24:])*prime64_2) * prime64_1
	}
	h.v1, h.v2, h.v3, h.v4 = v1, v2, v3, v4

	copy(h.buf[h.bufused:], input[p:])
	h.bufused += len(input) - p
}

// Write8 adds input bytes to the output hash without mutating the hash.
func Write8(h *Hash, input [8]byte) {
	n := len(input)
	m := h.bufused

	h.totalLen += uint64(n)

	r := len(h.buf) - m
	if n < r {
		copy(h.buf[m:], input[:])
		h.bufused += len(input)
		return
	}

	p := 0
	if m > 0 {
		// some data left from previous update
		copy(h.buf[h.bufused:], input[:r])
		h.bufused += len(input) - r

		// fast rotl(31)
		h.v1 = rol31(h.v1+u64(h.buf[:])*prime64_2) * prime64_1
		h.v2 = rol31(h.v2+u64(h.buf[8:])*prime64_2) * prime64_1
		h.v3 = rol31(h.v3+u64(h.buf[16:])*prime64_2) * prime64_1
		h.v4 = rol31(h.v4+u64(h.buf[24:])*prime64_2) * prime64_1
		p = r
		h.bufused = 0
	}

	// Causes compiler to work directly from registers instead of stack:
	v1, v2, v3, v4 := h.v1, h.v2, h.v3, h.v4
	for n := n - 32; p <= n; p += 32 {
		sub := input[p:][:32] // BCE hint for compiler
		v1 = rol31(v1+u64(sub[:])*prime64_2) * prime64_1
		v2 = rol31(v2+u64(sub[8:])*prime64_2) * prime64_1
		v3 = rol31(v3+u64(sub[16:])*prime64_2) * prime64_1
		v4 = rol31(v4+u64(sub[24:])*prime64_2) * prime64_1
	}
	h.v1, h.v2, h.v3, h.v4 = v1, v2, v3, v4

	copy(h.buf[h.bufused:], input[p:])
	h.bufused += len(input) - p
}

// Sum64 returns the 64 bit hash value.
func (h *Hash) Sum64() uint64 {
	var h64 uint64
	if h.totalLen >= 32 {
		h64 = rol1(h.v1) + rol7(h.v2) + rol12(h.v3) + rol18(h.v4)

		// h.v1 *= prime64_2
		// h.v2 *= prime64_2
		// h.v3 *= prime64_2
		// h.v4 *= prime64_2

		h64 = (h64^(rol31(h.v1)*prime64_1))*prime64_1 + prime64_4
		h64 = (h64^(rol31(h.v2)*prime64_1))*prime64_1 + prime64_4
		h64 = (h64^(rol31(h.v3)*prime64_1))*prime64_1 + prime64_4
		h64 = (h64^(rol31(h.v4)*prime64_1))*prime64_1 + prime64_4

		h64 += h.totalLen
	} else {
		h64 = h.seed + prime64_5 + h.totalLen
	}

	p := 0
	n := h.bufused
	for n := n - 8; p <= n; p += 8 {
		h64 ^= rol31(u64(h.buf[p:p+8])*prime64_2) * prime64_1
		h64 = rol27(h64)*prime64_1 + prime64_4
	}
	if p+4 <= n {
		sub := h.buf[p : p+4]
		h64 ^= uint64(u32(sub)) * prime64_1
		h64 = rol23(h64)*prime64_2 + prime64_3
		p += 4
	}
	for ; p < n; p++ {
		h64 ^= uint64(h.buf[p]) * prime64_5
		h64 = rol11(h64) * prime64_1
	}

	h64 ^= h64 >> 33
	h64 *= prime64_2
	h64 ^= h64 >> 29
	h64 *= prime64_3
	h64 ^= h64 >> 32

	return h64
}

func u64[B []byte | string](buf B) uint64 {
	// go compiler recognizes this pattern
	// and optimizes it on little endian platforms
	return uint64(buf[0]) |
		uint64(buf[1])<<8 |
		uint64(buf[2])<<16 |
		uint64(buf[3])<<24 |
		uint64(buf[4])<<32 |
		uint64(buf[5])<<40 |
		uint64(buf[6])<<48 |
		uint64(buf[7])<<56
}

func u32[B []byte | string](buf B) uint32 {
	return uint32(buf[0]) |
		uint32(buf[1])<<8 |
		uint32(buf[2])<<16 |
		uint32(buf[3])<<24
}

func rol1(u uint64) uint64  { return u<<1 | u>>63 }
func rol7(u uint64) uint64  { return u<<7 | u>>57 }
func rol11(u uint64) uint64 { return u<<11 | u>>53 }
func rol12(u uint64) uint64 { return u<<12 | u>>52 }
func rol18(u uint64) uint64 { return u<<18 | u>>46 }
func rol23(u uint64) uint64 { return u<<23 | u>>41 }
func rol27(u uint64) uint64 { return u<<27 | u>>37 }
func rol31(u uint64) uint64 { return u<<31 | u>>33 }
