package playmon

import (
	"fmt"

	"github.com/graph-guard/ggproxy/config"
	"github.com/graph-guard/ggproxy/engine/playmon/internal/constrcheck"
	"github.com/graph-guard/ggproxy/engine/playmon/internal/pathmatch"
	"github.com/graph-guard/ggproxy/engine/playmon/internal/pathscan"
	"github.com/graph-guard/ggproxy/gqlparse"
	"github.com/graph-guard/gqlscan"
	"github.com/graph-guard/gqt/v4"
)

type Engine struct {
	pathScanner *pathscan.PathScanner
	matcher     *pathmatch.Matcher
	templates   []*template

	structuralPaths [][]byte
	gqtVarPaths     map[string]int
	inputsByPath    map[string][]gqlparse.Token
}

type template struct {
	Index             int
	ID                string
	GQTOpr            *gqt.Operation
	ConstraintChecker *constrcheck.Checker
}

// New expects templates to be initialized and valid.
func New(s *config.Service) *Engine {
	e := &Engine{
		pathScanner: pathscan.New(128, 2048),
		templates:   make([]*template, len(s.Templates)),
		gqtVarPaths: make(map[string]int),

		inputsByPath: make(map[string][]gqlparse.Token),
	}
	idCounter := 0
	for _, t := range s.Templates {
		c := constrcheck.New(t.GQTTemplate, s.Schema)
		tmpl := &template{
			Index:             idCounter,
			ID:                t.ID,
			GQTOpr:            t.GQTTemplate,
			ConstraintChecker: c,
		}
		e.templates[idCounter] = tmpl
		pathscan.InAST(
			tmpl.GQTOpr,
			func(path []byte, e gqt.Expression) (stop bool) {
				return false
			}, func(path []byte, _ gqt.Expression) (stop bool) {
				return false
			}, func(path []byte, _ gqt.Expression) (stop bool) {
				e.gqtVarPaths[string(path)] = 0
				return false
			},
		)
		idCounter++
	}
	e.matcher = pathmatch.New(s)
	return e
}

func (e *Engine) reset() {
	e.structuralPaths = e.structuralPaths[:0]
	for i := range e.gqtVarPaths {
		e.gqtVarPaths[i] = 0
	}
	for path := range e.inputsByPath {
		e.gqtVarPaths[path] = 0
	}
}

// Match returns the ID of the first matching template or "" if none was matched.
func (e *Engine) Match(
	variableValues [][]gqlparse.Token,
	queryType gqlscan.Token,
	selectionSet []gqlparse.Token,
) (id string) {
	e.reset()
	fmt.Println(selectionSet)
	e.pathScanner.InTokens(
		queryType,
		selectionSet,
		e.gqtVarPaths,
		func(path []byte) (stop bool) {
			e.structuralPaths = append(e.structuralPaths, path)
			return false
		},
		func(path []byte, i int) (stop bool) {
			return false
		},
		func(path []byte, i int) (stop bool) {
			fmt.Println("FUCK")
			e.inputsByPath[string(path)] = selectionSet[i:]
			return false
		},
	)
	fmt.Println("STR", e.structuralPaths)
	for i := range e.structuralPaths {
		fmt.Println("  ", string(e.structuralPaths[i]))
	}

	e.matcher.Match(e.structuralPaths, func(t *config.Template) (stop bool) {
		fmt.Println("STRUCTURAL MATCH ", t.ID)
		fmt.Println(e.inputsByPath)
		id = t.ID
		return false
	})
	// e.currentMask.Visit(func(i int) (skip bool) {
	// 	t := e.templates[i]
	// 	match := true
	// 	t.ConstraintChecker.VisitPaths(func(path string) (stop bool) {
	// 		if !t.ConstraintChecker.Check(path) {
	// 			match = false
	// 			return true
	// 		}
	// 		return false
	// 	})
	// 	if match {
	// 		id = t.ID
	// 		return true
	// 	}
	// 	return false
	// })
	return id
}

// MatchAll calls fn for every matching template.
func (e *Engine) MatchAll(
	variableValues [][]gqlparse.Token,
	queryType gqlscan.Token,
	selectionSet []gqlparse.Token,
	fn func(id string),
) {
	panic("todo")
}
