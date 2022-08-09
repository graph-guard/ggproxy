package set_test

import (
	"encoding/binary"
	"fmt"
	"math/rand"
	"testing"

	"github.com/graph-guard/gguard-proxy/utilities/set"
)

var GB bool

func BenchmarkEnableDisableUint64(b *testing.B) {
	for _, td := range []struct {
		size   int
		enable int
	}{
		{8, 8},
		{64, 64},
		{512, 512},
	} {
		b.Run(fmt.Sprintf("%v", td), func(b *testing.B) {
			dataSet := make([]uint64, td.size)
			for i := range dataSet {
				dataSet[i] = RandUint64()
			}
			s := set.New(dataSet...)
			enable := dataSet[:td.enable]
			b.ResetTimer()

			for n := 0; n < b.N; n++ {
				for _, v := range enable {
					GB = s.Enable(v)
				}
				for _, v := range enable {
					GB = s.Disable(v)
				}
				s.Reset()
			}
		})
	}
}

var GI int

func BenchmarkCountEnabled(b *testing.B) {
	for _, td := range []struct {
		size   int
		enable int
	}{
		{8, 8},
		{64, 64},
		{512, 512},
	} {
		b.Run(fmt.Sprintf("%v", td), func(b *testing.B) {
			dataSet := make([]uint64, td.size)
			for i := range dataSet {
				dataSet[i] = RandUint64()
			}
			s := set.New(dataSet...)
			enable := dataSet[:td.enable]
			b.ResetTimer()

			for n := 0; n < b.N; n++ {
				for _, v := range enable {
					GB = s.Enable(v)
				}
				s.VisitEnabled(func(uint64) (stop bool) {
					GI++
					return false
				})
				s.Reset()
			}
		})
	}
}

func RandUint64() uint64 {
	buf := make([]byte, 8)
	rand.Read(buf) // Always succeeds, no need to check error
	return binary.LittleEndian.Uint64(buf)
}
