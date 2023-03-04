package xxhash_test

import (
	"testing"

	"github.com/graph-guard/ggproxy/pkg/xxhash"

	"github.com/pierrec/xxHash/xxHash64"
)

var GI uint64

func BenchmarkOriginal(b *testing.B) {
	s1 := []byte("foobar")
	s2 := []byte("bazzfuzz")
	h := xxHash64.New(0)
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_, _ = h.Write(s1)
		_, _ = h.Write(s2)
		GI = h.Sum64()
		h.Reset()
	}
}

func BenchmarkCustom(b *testing.B) {
	s1 := []byte("foobar")
	s2 := []byte("bazzfuzz")
	h := xxhash.New(0)
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		xxhash.Write(&h, s1)
		xxhash.Write(&h, s2)
		GI = h.Sum64()
	}
}
