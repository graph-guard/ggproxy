package pathmatch

import (
	"github.com/graph-guard/ggproxy/pkg/bitmask"
	"github.com/graph-guard/ggproxy/pkg/config"
	"github.com/graph-guard/ggproxy/pkg/engine/playmon/internal/pathinfo"
	"github.com/graph-guard/ggproxy/pkg/engine/playmon/internal/pathscan"
	"github.com/graph-guard/gqt/v4"
)

type structuralPath struct {
	Name        string
	Mask        *bitmask.Set
	Combinators []combination
}

type combination struct {
	Index         int
	Depth         int
	TemplateIndex int
}

type template struct {
	*config.Template
	paths []uint64
}

type Matcher struct {
	conf      *config.Service
	templates []template
	paths     map[uint64]structuralPath

	// combinatorsLimits defines the limit of the `max` set of every combination.
	combinatorsLimits []int

	lengths []int

	/* Operational data, reset for every match call */

	// combinatorCounters keeps the counters for every combination.
	combinatorCounters []int

	matchMask, rejectedMask *bitmask.Set
	matchesPerTemplate      []int
}

func (m *Matcher) reset() {
	m.matchMask.Reset()
	m.rejectedMask.Reset()
	for i := range m.matchesPerTemplate {
		m.matchesPerTemplate[i] = 0
	}
	for i := range m.combinatorCounters {
		m.combinatorCounters[i] = 0
	}
}

// Match calls onMatch for every template matching paths.
func (m *Matcher) Match(
	paths []uint64,
	onMatch func(*config.Template) (stop bool),
) {
	m.reset()

	for i := range paths {
		b, ok := m.paths[paths[i]]
		if !ok {
			return // Unknown path, can't match any template
		}

		for _, c := range b.Combinators {
			depth := 0
			if m.combinatorCounters[c.Index] < 1 {
				depth = c.Depth
			}
			for i := c.Index - depth; i <= c.Index; i++ {
				m.combinatorCounters[i]++
				if m.combinatorsLimits[i] < m.combinatorCounters[i] {
					m.rejectedMask.Add(c.TemplateIndex)
				}
			}
		}

		b.Mask.VisitAll(func(n int) { m.matchesPerTemplate[n]++ })
		m.matchMask.SetOr(m.matchMask, b.Mask)
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
		conf:      conf,
		paths:     make(map[uint64]structuralPath, len(conf.TemplatesEnabled)),
		templates: make([]template, len(conf.TemplatesEnabled)),
		lengths:   make([]int, len(conf.TemplatesEnabled)),

		matchMask:          bitmask.New(),
		rejectedMask:       bitmask.New(),
		matchesPerTemplate: make([]int, len(conf.TemplatesEnabled)),
	}
	var maxCombinators []*gqt.SelectionMax
	for i := range conf.TemplatesEnabled {
		if errs := pathscan.InAST(
			conf.TemplatesEnabled[i].GQTTemplate,
			func(
				path string,
				pathHash uint64,
				e gqt.Expression,
			) (stop bool) {
				// On structural
				m.templates[i].paths = append(m.templates[i].paths, pathHash)

				var v structuralPath
				var ok bool
				if v, ok = m.paths[pathHash]; !ok {
					v.Mask = bitmask.New()
					v.Name = path
					m.paths[pathHash] = v
				}

				if depth, parentMax := pathinfo.Info(e); depth > 0 {
					index := indexOf(maxCombinators, parentMax)
					if index < 0 {
						index = len(maxCombinators)
						maxCombinators = append(maxCombinators, parentMax)
						m.combinatorsLimits = append(
							m.combinatorsLimits, parentMax.Limit,
						)
						m.combinatorCounters = append(m.combinatorCounters, 0)
					}

					// Register combinator for paths inside `max` sets
					v.Combinators = append(v.Combinators, combination{
						Index:         index,
						Depth:         depth - 1,
						TemplateIndex: i,
					})
					m.paths[pathHash] = v
				}
				return false
			},
			func(
				path string,
				pathHash uint64,
				e gqt.Expression,
			) (stop bool) {
				// On argument
				return false
			},
			func(
				path string,
				pathHash uint64,
				e *gqt.VariableDeclaration,
			) (stop bool) {
				// On variable
				return false
			},
		); errs != nil {
			panic(errs)
		}

		// Initialize path bitmasks
		for _, p := range m.templates[i].paths {
			m.paths[p].Mask.Add(i)
		}
		m.lengths[i] = len(m.templates[i].paths)
	}

	return m
}

func indexOf[T comparable](s []T, x T) (index int) {
	for i := range s {
		if s[i] == x {
			return i
		}
	}
	return -1
}
