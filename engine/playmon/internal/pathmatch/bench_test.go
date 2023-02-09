package pathmatch_test

import (
	"testing"

	"github.com/graph-guard/ggproxy/config"
)

var GI int

func BenchmarkPathmatch(b *testing.B) {
	for _, bb := range tests {
		b.Run(bb.name, func(b *testing.B) {
			m := prepareTestSetup(b, bb.conf)
			b.ResetTimer()
			for n := 0; n < b.N; n++ {
				m.Match(bb.paths, func(tm *config.Template) {
					GI++
				})
			}
		})
	}
}
