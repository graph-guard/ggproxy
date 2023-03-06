package main

import "github.com/graph-guard/ggproxy/pkg/bitmask"

type (
	Variant struct {
		Mask         *bitmask.Set
		Combinations []Combination
	}
	Combination struct{ CombinationIndex, Depth, TemplateIndex int }
)

/*
	A: query {
		max 1 {
			a {
				max 1 {
					a_0
					a_1
				}
			}
			b
		}
	}
	B: query {
		max 2 {
			a {
				max 2 {
					a_0
					a_1
					a_2
				}
			}
			b
			c
		}
	}
*/

// combinations is sorted by the order of encountered paths while
// parsing the templates.
var combinations = []Combination{
	{CombinationIndex: 0, Depth: 1, TemplateIndex: 0}, // A: Q.a.a_0
	{CombinationIndex: 1, Depth: 1, TemplateIndex: 0}, // A: Q.a.a_1
	{CombinationIndex: 2, Depth: 0, TemplateIndex: 0}, // A: Q.b
	{CombinationIndex: 3, Depth: 1, TemplateIndex: 1}, // B: Q.a.a_0
	{CombinationIndex: 4, Depth: 1, TemplateIndex: 1}, // B: Q.a.a_1
	{CombinationIndex: 5, Depth: 1, TemplateIndex: 1}, // B: Q.a.a_2
	{CombinationIndex: 6, Depth: 0, TemplateIndex: 1}, // B: Q.b
	{CombinationIndex: 7, Depth: 0, TemplateIndex: 1}, // B: Q.c
}

// limits define the limit of the max combinator across all combinations.
var limits = []int{
	1, // A: Q.a.a_0
	1, // A: Q.a.a_1
	1, // A: Q.b
	2, // B: Q.a.a_0
	2, // B: Q.a.a_1
	2, // B: Q.a.a_2
	2, // B: Q.b
	2, // B: Q.c
}

// counters is reset on every matching attempt.
var counters = []int{
	2, // A: Q.a.a_0
	1, // A: Q.a.a_1
	1, // A: Q.b
	1, // B: Q.a.a_0
	1, // B: Q.a.a_1
	0, // B: Q.a.a_2
	1, // B: Q.b
	0, // B: Q.c
}

// query { b a { a_1 a_0 } }

var structural = map[string][]Variant{
	"Q.a.a_0": {{
		Mask:         bitmask.New(0, 1),
		Combinations: combinations[0:1],
	}, {
		Mask:         bitmask.New(0, 1),
		Combinations: combinations[3:4],
	}},
	"Q.a.a_1": {{
		Mask:         bitmask.New(0, 1),
		Combinations: combinations[1:2],
	}, {
		Mask:         bitmask.New(0, 1),
		Combinations: combinations[4:5],
	}},
	"Q.a.a_2": {{
		Mask:         bitmask.New(1),
		Combinations: combinations[5:6],
	}},
	"Q.b": {{
		Mask:         bitmask.New(0, 1),
		Combinations: combinations[2:3],
	}, {
		Mask:         bitmask.New(0, 1),
		Combinations: combinations[6:7],
	}},
	"Q.c": {{
		Mask:         bitmask.New(1),
		Combinations: combinations[7:8],
	}},
}

func main() {
	for i := range counters {
		counters[i] = 0
	}

	for _, v := range structural["Q.a"] {
		for _, c := range v.Combinations {
			for i := maxInt(0, c.CombinationIndex-c.Depth); i <= c.CombinationIndex; i++ {
				counters[i]++
				if limits[i] < counters[i] {
					// reject c.TemplateIndex
				}
			}
		}
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
