package graph_test

import (
	"strconv"
	"testing"

	"github.com/graph-guard/gguard-proxy/gqlreduce/internal/graph"
	"github.com/graph-guard/gguard-proxy/utilities/decl"
	"github.com/stretchr/testify/require"
)

func TestMake(t *testing.T) {
	for _, td := range testdata {
		t.Run(td.Decl, func(t *testing.T) {
			d := graph.NewInspector()
			var cyclePath, ordered []string
			le := d.Make(
				td.Data.Graph,
				func(nodeName []byte) {
					cyclePath = append(cyclePath, string(nodeName))
				},
				func(nodeName []byte) {
					ordered = append(ordered, string(nodeName))
				},
			)
			require.False(t, le)
			require.Len(t, cyclePath, 0)
			require.Equal(t, td.Data.ExpectOrder, ordered)
		})
	}
}

func TestCycle(t *testing.T) {
	for _, td := range testdataCyclic {
		t.Run(td.Decl, func(t *testing.T) {
			d := graph.NewInspector()
			var cyclePath []string
			le := d.Make(
				td.Data.Graph,
				func(nodeName []byte) {
					cyclePath = append(cyclePath, string(nodeName))
				},
				func(nodeName []byte) {},
			)
			require.False(t, le)
			require.Equal(t, td.Data.ExpectCyclePath, cyclePath)
		})
	}
}

func TestLimit(t *testing.T) {
	d := graph.NewInspector()
	e := make([]graph.Edge, 0, graph.MaxFragments)
	for i := 0; i < graph.MaxFragments-1; i++ {
		from := []byte(strconv.Itoa(i))
		to := []byte(strconv.Itoa(i + 1))
		e = append(e, graph.Edge{from, to})
	}

	var cyclePath, ordered []string
	e = append(e, graph.Edge{[]byte("0"), []byte("a")})
	le := d.Make(
		e,
		func(nodeName []byte) {
			cyclePath = append(cyclePath, string(nodeName))
		},
		func(nodeName []byte) {
			ordered = append(ordered, string(nodeName))
		},
	)
	require.True(t, le)
	require.Len(t, cyclePath, 0)
	require.Len(t, ordered, 0)
}

type TestOK struct {
	Graph       []graph.Edge
	ExpectOrder []string
}

var testdata = []decl.Declaration[TestOK]{
	decl.New(TestOK{
		Graph:       []graph.Edge{},
		ExpectOrder: nil,
	}),
	decl.New(TestOK{
		/*
			digraph G {
				f -> f2
			}
		*/
		Graph: []graph.Edge{
			{From: []byte("f"), To: []byte("f2")},
		},
		ExpectOrder: []string{
			"f2", "f",
		},
	}),
	decl.New(TestOK{
		/*
			digraph G {
				f -> f2; f -> f3
				f3 -> f2
			}
		*/
		Graph: []graph.Edge{
			{From: []byte("f"), To: []byte("f2")},
			{From: []byte("f"), To: []byte("f3")},
			{From: []byte("f3"), To: []byte("f2")},
		},
		ExpectOrder: []string{
			"f2", "f3", "f",
		},
	}),
	decl.New(TestOK{
		/*
			digraph G {
				f -> f2; f -> fA
				f2 -> f3; f2 -> fB
				f3 -> f4
				fA -> fB
				fB -> fC
				fX -> fA
			}
		*/
		Graph: []graph.Edge{
			{From: []byte("f"), To: []byte("f2")},
			{From: []byte("f"), To: []byte("fA")},
			{From: []byte("f2"), To: []byte("f3")},
			{From: []byte("f2"), To: []byte("fB")},
			{From: []byte("f3"), To: []byte("f4")},
			{From: []byte("fA"), To: []byte("fB")},
			{From: []byte("fB"), To: []byte("fC")},
			{From: []byte("fX"), To: []byte("fA")},
		},
		ExpectOrder: []string{
			"f4", "f3", "fC", "fB", "f2", "fA", "f", "fX",
		},
	}),
}

type CycleTest struct {
	Graph           []graph.Edge
	ExpectCyclePath []string
}

var testdataCyclic = []decl.Declaration[CycleTest]{
	// Cycles
	decl.New(CycleTest{
		// fB->fC->fA->fB
		Graph: []graph.Edge{
			{From: []byte("f"), To: []byte("f2")},
			{From: []byte("f"), To: []byte("fX")},
			{From: []byte("f2"), To: []byte("f3")},
			{From: []byte("f2"), To: []byte("fB")},
			{From: []byte("f3"), To: []byte("f4")},
			{From: []byte("fA"), To: []byte("fB")},
			{From: []byte("fB"), To: []byte("fC")},
			{From: []byte("fC"), To: []byte("fA")},
		},
		ExpectCyclePath: []string{"fB", "fC", "fA", "fB"},
	}),
	decl.New(CycleTest{
		// f->f
		Graph: []graph.Edge{
			{From: []byte("f"), To: []byte("f")},
		},
		ExpectCyclePath: []string{"f", "f"},
	}),
	decl.New(CycleTest{
		// c->c
		Graph: []graph.Edge{
			{From: []byte("a"), To: []byte("b")},
			{From: []byte("c"), To: []byte("c")},
		},
		ExpectCyclePath: []string{"c", "c"},
	}),
	decl.New(CycleTest{
		// f3->f->f2->f3
		Graph: []graph.Edge{
			{From: []byte("f3"), To: []byte("f")},
			{From: []byte("f2"), To: []byte("f3")},
			{From: []byte("f"), To: []byte("f2")},
		},
		ExpectCyclePath: []string{"f3", "f", "f2", "f3"},
	}),
	decl.New(CycleTest{
		// f->f2->f
		Graph: []graph.Edge{
			{From: []byte("f"), To: []byte("f2")},
			{From: []byte("f2"), To: []byte("f")},
		},
		ExpectCyclePath: []string{"f", "f2", "f"},
	}),
	decl.New(CycleTest{
		// f->f2->f3->f
		Graph: []graph.Edge{
			{From: []byte("f"), To: []byte("f2")},
			{From: []byte("f2"), To: []byte("f3")},
			{From: []byte("f3"), To: []byte("f")},
		},
		ExpectCyclePath: []string{"f", "f2", "f3", "f"},
	}),
	decl.New(CycleTest{
		// f->f2->f3->f
		Graph: []graph.Edge{
			{From: []byte("f"), To: []byte("f2")},
			{From: []byte("f"), To: []byte("fA")},
			{From: []byte("f2"), To: []byte("f3")},
			{From: []byte("f2"), To: []byte("fB")},
			{From: []byte("f3"), To: []byte("f")},
			{From: []byte("fA"), To: []byte("fB")},
			{From: []byte("fB"), To: []byte("fC")},
			{From: []byte("fX"), To: []byte("fA")},
		},
		ExpectCyclePath: []string{"f", "f2", "f3", "f"},
	}),
	decl.New(CycleTest{
		// alpha->...->oscar->alpha
		Graph: []graph.Edge{
			{From: []byte("alpha"), To: []byte("a1")},
			{From: []byte("alpha"), To: []byte("a2")},
			{From: []byte("bravo"), To: []byte("b2")},
			{From: []byte("bravo"), To: []byte("b1")},
			{From: []byte("charlie"), To: []byte("c2")},
			{From: []byte("charlie"), To: []byte("c1")},
			{From: []byte("delta"), To: []byte("d2")},
			{From: []byte("delta"), To: []byte("d1")},
			{From: []byte("echo"), To: []byte("e2")},
			{From: []byte("echo"), To: []byte("e1")},
			{From: []byte("foxtrot"), To: []byte("f2")},
			{From: []byte("foxtrot"), To: []byte("f1")},
			{From: []byte("golf"), To: []byte("g2")},
			{From: []byte("golf"), To: []byte("g1")},
			{From: []byte("hotel"), To: []byte("h2")},
			{From: []byte("hotel"), To: []byte("h1")},
			{From: []byte("india"), To: []byte("i1")},
			{From: []byte("india"), To: []byte("i2")},
			{From: []byte("juliette"), To: []byte("j2")},
			{From: []byte("juliette"), To: []byte("j1")},
			{From: []byte("kilo"), To: []byte("k2")},
			{From: []byte("kilo"), To: []byte("k1")},
			{From: []byte("lima"), To: []byte("l2")},
			{From: []byte("lima"), To: []byte("l1")},
			{From: []byte("mike"), To: []byte("m2")},
			{From: []byte("mike"), To: []byte("m1")},
			{From: []byte("november"), To: []byte("n2")},
			{From: []byte("november"), To: []byte("n1")},
			{From: []byte("oscar"), To: []byte("o2")},
			{From: []byte("oscar"), To: []byte("o1")},

			{From: []byte("alpha"), To: []byte("bravo")},
			{From: []byte("bravo"), To: []byte("charlie")},
			{From: []byte("charlie"), To: []byte("delta")},
			{From: []byte("delta"), To: []byte("echo")},
			{From: []byte("echo"), To: []byte("foxtrot")},
			{From: []byte("foxtrot"), To: []byte("golf")},
			{From: []byte("golf"), To: []byte("hotel")},
			{From: []byte("hotel"), To: []byte("india")},
			{From: []byte("india"), To: []byte("juliette")},
			{From: []byte("juliette"), To: []byte("kilo")},
			{From: []byte("kilo"), To: []byte("lima")},
			{From: []byte("lima"), To: []byte("mike")},
			{From: []byte("mike"), To: []byte("november")},
			{From: []byte("november"), To: []byte("oscar")},
			{From: []byte("oscar"), To: []byte("alpha")},
		},
		ExpectCyclePath: []string{
			"alpha", "bravo", "charlie", "delta",
			"echo", "foxtrot", "golf", "hotel",
			"india", "juliette", "kilo", "lima",
			"mike", "november", "oscar", "alpha",
		},
	}),
}

type ChildrenTest struct {
	Graph  []graph.Edge
	Expect map[string][]string
}

var testdataChildren = []decl.Declaration[ChildrenTest]{
	decl.New(ChildrenTest{
		/*
			digraph G {
				f -> f2
			}
		*/
		Graph: []graph.Edge{
			{From: []byte("f"), To: []byte("f2")},
		},
		Expect: map[string][]string{
			"f":  {"f2"},
			"f2": nil,
		},
	}),
	decl.New(ChildrenTest{
		/*
			digraph G {
				f -> f2; f -> f3
				f3 -> f2
			}
		*/
		Graph: []graph.Edge{
			{From: []byte("f"), To: []byte("f2")},
			{From: []byte("f"), To: []byte("f3")},
			{From: []byte("f3"), To: []byte("f2")},
		},
		Expect: map[string][]string{
			"f":  {"f2", "f3"},
			"f3": {"f2"},
			"f2": nil,
		},
	}),
	decl.New(ChildrenTest{
		/*
			digraph G {
				f -> f2; f -> fA
				f2 -> f3; f2 -> fB
				f3 -> f4
				fA -> fB
				fB -> fC
				fX -> fA
			}
		*/
		Graph: []graph.Edge{
			{From: []byte("f"), To: []byte("f2")},
			{From: []byte("f"), To: []byte("fA")},
			{From: []byte("f2"), To: []byte("f3")},
			{From: []byte("f2"), To: []byte("fB")},
			{From: []byte("f3"), To: []byte("f4")},
			{From: []byte("fA"), To: []byte("fB")},
			{From: []byte("fB"), To: []byte("fC")},
			{From: []byte("fX"), To: []byte("fA")},
		},
		Expect: map[string][]string{
			"f":  {"f2", "fA"},
			"f2": {"f3", "fB"},
			"f3": {"f4"},
			"fA": {"fB"},
			"fB": {"fC"},
			"fX": {"fA"},
			"f4": nil,
			"fC": nil,
		},
	}),
}

func TestVisitChildren(t *testing.T) {
	for _, td := range testdataChildren {
		t.Run(td.Decl, func(t *testing.T) {
			d := graph.NewInspector()
			var ordered []string
			var cyclePath []string
			le := d.Make(
				td.Data.Graph,
				func(nodeName []byte) {
					cyclePath = append(cyclePath, string(nodeName))
				},
				func(nodeName []byte) {
					ordered = append(ordered, string(nodeName))
				},
			)
			require.False(t, le)
			require.Len(t, cyclePath, 0)

			actual := map[string][]string{}
			for _, n := range ordered {
				children := []string{}
				d.VisitChildren([]byte(n), func(b []byte) {
					children = append(children, string(b))
				})
				actual[n] = children
			}

			require.Len(t, actual, len(td.Data.Expect))
			for k, v := range td.Data.Expect {
				require.Contains(t, actual, k)
				require.Equal(t, td.Data.Expect[k], v)
			}
		})
	}
}
