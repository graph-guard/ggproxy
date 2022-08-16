package aset_test

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"testing"

	"github.com/graph-guard/ggproxy/utilities/aset"
)

func BenchmarkAdd(b *testing.B) {
	for _, td := range []int{
		8, 64, 512,
	} {
		b.Run(fmt.Sprintf("%v", td), func(b *testing.B) {
			dataSet := make([]uint64, td)
			for i := range dataSet {
				dataSet[i] = RandUint64()
			}
			s := aset.New[uint64](1024)
			b.ResetTimer()

			for n := 0; n < b.N; n++ {
				for _, v := range dataSet {
					s.Add(v)
				}
			}
		})
	}
}

func BenchmarkFind(b *testing.B) {
	for _, td := range []int{
		8, 64, 512,
	} {
		b.Run(fmt.Sprintf("%v", td), func(b *testing.B) {
			dataSet := make([]uint64, td)
			for i := range dataSet {
				dataSet[i] = RandUint64()
			}
			s := aset.New(1024, dataSet...)
			el := RandUint64()
			b.ResetTimer()

			for n := 0; n < b.N; n++ {
				s.Find(el)
			}
		})
	}
}

func BenchmarkLen(b *testing.B) {
	for _, td := range []int{
		8, 64, 512,
	} {
		b.Run(fmt.Sprintf("%v", td), func(b *testing.B) {
			dataSet := make([]uint64, td)
			for i := range dataSet {
				dataSet[i] = RandUint64()
			}
			s := aset.New(1024, dataSet...)
			b.ResetTimer()

			for n := 0; n < b.N; n++ {
				s.Len()
			}
		})
	}
}

func RandUint64() uint64 {
	buf := make([]byte, 8)
	_, _ = rand.Read(buf) // Always succeeds, no need to check error

	return binary.LittleEndian.Uint64(buf)
}
