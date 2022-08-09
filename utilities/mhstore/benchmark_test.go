package mhstore_test

import (
	"fmt"
	"testing"

	"github.com/graph-guard/gguard-proxy/utilities/mhstore"
)

func BenchmarkAdd(b *testing.B) {
	for _, td := range []struct {
		masks  int
		hashes int
	}{
		{2, 8},
		{8, 8},
		{32, 32},
		{32, 384},
		{128, 384},
		{384, 8},
	} {
		b.Run(fmt.Sprintf("%v", td), func(b *testing.B) {
			masks := map[uint16]bool{}
			for len(masks) < td.masks {
				masks[RandUint16()] = true
			}
			hashes := map[uint64]bool{}
			for len(hashes) < td.hashes {
				hashes[RandUint64()] = true
			}
			store := mhstore.New()
			b.ResetTimer()

			for n := 0; n < b.N; n++ {
				for m := range masks {
					for h := range hashes {
						store.Add(m, h)
					}
				}
			}
		})
	}
}
