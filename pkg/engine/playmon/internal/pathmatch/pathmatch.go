package pathmatch

import (
	"fmt"

	"github.com/graph-guard/ggproxy/pkg/bitmask"
	"github.com/graph-guard/ggproxy/pkg/config"
	"github.com/graph-guard/ggproxy/pkg/engine/playmon/internal/pathinfo"
	"github.com/graph-guard/ggproxy/pkg/engine/playmon/internal/pathscan"
	"github.com/graph-guard/ggproxy/pkg/math"
	"github.com/graph-guard/gqt/v4"
)

type structuralPath struct {
	Name         string
	Mask         *bitmask.Set
	Combinations []combination
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

	// maxSetLimits defines the limit of the `max` set of every combination.
	maxSetLimits []int

	lengths []int

	/* Operational data, reset for every match call */

	// maxCounters keeps the counters for every combination.
	maxCounters []int

	matchMask, rejectedMask *bitmask.Set
	matchesPerTemplate      []int
}

func (m *Matcher) reset() {
	m.matchMask.Reset()
	m.rejectedMask.Reset()
	for i := range m.matchesPerTemplate {
		m.matchesPerTemplate[i] = 0
	}
	for i := range m.maxCounters {
		m.maxCounters[i] = 0
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

		for _, c := range b.Combinations {
			for i := math.Max(0, c.Index-c.Depth); i <= c.Index; i++ {
				m.maxCounters[i]++
				if m.maxSetLimits[i] < m.maxCounters[i] {
					m.rejectedMask.Add(i)
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
	var paths, templateIDs []string
	var combinations []combination
	for i := range conf.TemplatesEnabled {
		{ // Associate paths with templates by index
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
						// Register combinations for paths inside `max` sets
						paths = append(paths, path)
						templateIDs = append(templateIDs, conf.TemplatesEnabled[i].ID)
						v.Combinations = append(v.Combinations, combination{
							Index:         len(m.maxSetLimits),
							Depth:         depth - 1,
							TemplateIndex: i,
						})
						combinations = append(combinations, combination{
							Index:         len(m.maxSetLimits),
							Depth:         depth - 1,
							TemplateIndex: i,
						})
						m.paths[pathHash] = v

						m.maxSetLimits = append(m.maxSetLimits, parentMax.Limit)
						m.maxCounters = append(m.maxCounters, 0)
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
		}

		// Initialize path bitmasks
		for _, p := range m.templates[i].paths {
			m.paths[p].Mask.Add(i)
		}
		m.lengths[i] = len(m.templates[i].paths)
	}

	/*DUMP*/
	fmt.Println("\n<################# ENGINE DUMP>")
	fmt.Println("paths: ", len(m.paths))
	for pathHash, p := range m.paths {
		fmt.Printf(" %s: %d\n", p.Name, pathHash)
		for _, c := range p.Combinations {
			fmt.Printf(
				" - CombinationIndex: %d; Depth: %d; Template: %d\n",
				c.Index, c.Depth, c.TemplateIndex,
			)
		}
	}
	fmt.Println("combinations: ", len(combinations))
	for i, c := range combinations {
		fmt.Printf(
			" - CombinationIndex: %d; Depth: %d; Template: %d // %s: %s\n",
			c.Index, c.Depth, c.TemplateIndex, templateIDs[i], paths[i],
		)
	}
	fmt.Println("maxSetLimits: ", len(m.maxSetLimits))
	for i := range m.maxSetLimits {
		fmt.Printf(" %d: %d\n", i, m.maxSetLimits[i])
	}
	fmt.Println("maxCounters: ", len(m.maxCounters))
	for i := range m.maxCounters {
		fmt.Printf(" %d: %d\n", i, m.maxCounters[i])
	}
	fmt.Println("<################# ENGINE DUMP/>")

	return m
}
