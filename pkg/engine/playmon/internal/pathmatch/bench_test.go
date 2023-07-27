package pathmatch_test

import (
	"testing"

	"github.com/graph-guard/ggproxy/pkg/config"
	"github.com/graph-guard/ggproxy/pkg/engine/playmon/internal/pathscan"
)

var GI int

func BenchmarkPathmatch(b *testing.B) {
	for _, bb := range tests {
		b.Run(bb.name, func(b *testing.B) {
			m := prepareTestSetup(b, bb.conf)
			for _, bb := range bb.tests {
				b.Run(bb.name, func(b *testing.B) {
					paths := make([]uint64, len(bb.paths))
					for i := range bb.paths {
						paths[i] = pathscan.Hash(bb.paths[i])
					}
					b.ResetTimer()
					for n := 0; n < b.N; n++ {
						m.Match(paths, func(tm *config.Template) (stop bool) {
							GI++
							return false
						})
					}
				})
			}
		})
	}
}
