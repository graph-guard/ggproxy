package pathmatch

import (
	"github.com/graph-guard/ggproxy/config"
	"github.com/graph-guard/ggproxy/engine/playmon/internal/pathscan"
	"github.com/graph-guard/ggproxy/utilities/bitmask"
	"github.com/graph-guard/gqt/v4"
)

type Matcher struct {
	conf          *config.Service
	templatePaths [][]uint64
	paths         map[uint64]*bitmask.Set
	lengths       []int

	// Operational data, reset for every match call
	matchMask, rejectedMask *bitmask.Set
	matchesPerTemplate      []int
}

// Match calls onMatch for every template matching paths
func (m *Matcher) Match(
	paths []uint64,
	onMatch func(*config.Template) (stop bool),
) {
	m.matchMask.Reset()
	m.rejectedMask.Reset()
	for i := range m.matchesPerTemplate {
		m.matchesPerTemplate[i] = 0
	}

	for i := range paths {
		b, ok := m.paths[paths[i]]
		if !ok {
			return // Unknown path, can't match any template
		}
		b.VisitAll(func(n int) { m.matchesPerTemplate[n]++ })
		m.matchMask.SetOr(m.matchMask, b)
	}
	for i := range m.matchesPerTemplate {
		if m.matchesPerTemplate[i] < len(paths) {
			m.rejectedMask.Add(i)
		}
	}
	m.matchMask.SetAndNot(m.matchMask, m.rejectedMask)
	m.matchMask.Visit(func(n int) (stop bool) {
		return onMatch(m.conf.TemplatesEnabled[n])
	})
}

func New(conf *config.Service) *Matcher {
	m := &Matcher{
		conf: conf,
		// There will at least be 1 path per template
		paths:         make(map[uint64]*bitmask.Set, len(conf.TemplatesEnabled)),
		templatePaths: make([][]uint64, len(conf.TemplatesEnabled)),
		lengths:       make([]int, len(conf.TemplatesEnabled)),

		matchMask:          bitmask.New(),
		rejectedMask:       bitmask.New(),
		matchesPerTemplate: make([]int, len(conf.TemplatesEnabled)),
	}
	for i := range conf.TemplatesEnabled {
		{ // Associate paths with templates by index
			if errs := pathscan.InAST(
				conf.TemplatesEnabled[i].GQTTemplate,
				func(pathHash uint64, e gqt.Expression) (stop bool) {
					// On structural
					m.templatePaths[i] = append(m.templatePaths[i], pathHash)
					return false
				},
				func(pathHash uint64, e gqt.Expression) (stop bool) {
					// On argument
					return false
				},
				func(pathHash uint64, e *gqt.VariableDeclaration) (stop bool) {
					// On variable
					return false
				},
			); errs != nil {
				panic(errs)
			}
		}

		// Initialize path bitmasks
		for _, p := range m.templatePaths[i] {
			if _, ok := m.paths[p]; !ok {
				m.paths[p] = bitmask.New()
			}
			m.paths[p].Add(i)
		}
		m.lengths[i] = len(m.templatePaths[i])
	}
	return m
}
