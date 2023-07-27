package graph_test

import (
	"testing"

	"github.com/graph-guard/ggproxy/pkg/gqlparse/internal/graph"
)

var (
	GL bool
	GB []byte
)

func BenchmarkIsCyclic(b *testing.B) {
	for _, td := range testdataCyclic {
		b.Run(td.Decl, func(b *testing.B) {
			d := graph.NewInspector()
			b.ResetTimer()

			for n := 0; n < b.N; n++ {
				GL = d.Make(
					td.Data.Graph,
					func(nodeName []byte) {
						// On cycle
						GB = nodeName
					},
					func(nodeName []byte) {
						// Ordered
						GB = nodeName
					},
				)
			}
		})
	}
}

func BenchmarkIndexCycleAll(b *testing.B) {
	d := graph.NewInspector()
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		for i := 0; i < len(testdataCyclic); i++ {
			GL = d.Make(
				testdataCyclic[i].Data.Graph,
				func(nodeName []byte) {
					// On cycle
					GB = nodeName
				},
				func(nodeName []byte) {
					// Ordered
					GB = nodeName
				},
			)
		}
	}
}
