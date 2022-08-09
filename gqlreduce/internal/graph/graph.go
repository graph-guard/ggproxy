package graph

import (
	"github.com/graph-guard/gguard-proxy/utilities/container/hamap"
	"github.com/graph-guard/gguard-proxy/utilities/stack"
)

const MaxFragments = 128

func NewInspector() *Inspector {
	return &Inspector{
		ni:    hamap.New[[]byte, int](MaxFragments, nil),
		stack: stack.New[int](MaxFragments),
	}
}

type Inspector struct {
	ni    *hamap.Map[[]byte, int]
	stack *stack.Stack[int]
	n     int
	cl    [MaxFragments]uint8
	g     [MaxFragments][MaxFragments]bool
	names [MaxFragments][]byte
}

func (d *Inspector) reset() {
	d.ni.Reset()
	for i := range d.g[:d.n] {
		for j := range d.g[:d.n] {
			d.g[i][j] = false
		}
	}
	d.n = 0
}

type Edge struct{ From, To []byte }

// Make iterates over edges and forms the graph.
// Calls onCycle for every path element in case of a cycle.
// Calls ordered for every node of the graph if the graph.
func (d *Inspector) Make(
	edges []Edge,
	onCyclePathElement func(nodeName []byte),
	ordered func(nodeName []byte),
) (limitExceeded bool) {
	d.reset()
	for i := 0; i < len(edges); i++ {
		var fromIdx, toIdx int
		var ok bool
		if fromIdx, ok = d.ni.Get(edges[i].From); !ok {
			if d.n+1 > MaxFragments {
				return true
			}
			d.ni.Set(edges[i].From, d.n)
			fromIdx = d.n
			d.n++
			d.names[fromIdx] = edges[i].From
		}
		if toIdx, ok = d.ni.Get(edges[i].To); !ok {
			if d.n+1 > MaxFragments {
				return true
			}
			d.ni.Set(edges[i].To, d.n)
			toIdx = d.n
			d.n++
			d.names[toIdx] = edges[i].To
		}
		d.g[fromIdx][toIdx] = true
	}

	for i := range d.cl {
		d.cl[i] = 0
	}

	for i := 0; i < d.n; i++ {
		if d.cl[i] != 0 {
			continue
		}
		d.stack.Reset()
		from, to := i, 0
	S:
		d.stack.Push(from)
		d.cl[from] = 1
	E:
		for j := to; j < d.n; j++ {
			if !d.g[from][j] {
				continue
			} else if d.cl[j] == 0 {
				from, to = j, 0
				goto S
			} else if d.cl[j] == 1 {
				// Cycle detected
				for x := 0; x < d.stack.Len(); x++ {
					if d.stack.Get(x) != j {
						continue
					}
					for x := x; x < d.stack.Len(); x++ {
						onCyclePathElement(d.names[d.stack.Get(x)])
					}
					break
				}
				onCyclePathElement(d.names[j])
				return false
			}
		}
		d.cl[from], to = 2, from+1
		ordered(d.names[from])
		d.stack.Pop()
		if d.stack.Len() > 0 {
			from = d.stack.Top()
			goto E
		}
	}

	return false
}

// VisitChildren calls fn for every child of node indexed by name.
func (d *Inspector) VisitChildren(name []byte, fn func([]byte)) {
	p, ok := d.ni.Get(name)
	if !ok {
		return
	}
	for c := 0; c < d.n; c++ {
		if d.g[p][c] {
			fn(d.names[c])
		}
	}
}
