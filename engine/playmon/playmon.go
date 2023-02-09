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

	inputsByPath map[string][]gqlparse.Token
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

		inputsByPath: make(map[string][]gqlparse.Token),
	}
	idCounter := 0
	for _, t := range s.Templates {
		c := constrcheck.New(t.GQTTemplate, s.Schema)
		e.templates[idCounter] = &template{
			Index:             idCounter,
			ID:                t.ID,
			GQTOpr:            t.GQTTemplate,
			ConstraintChecker: c,
		}
		idCounter++
	}
	pathmatch.New(s)
	return e
}

// Match returns the ID of the first matching template or "" if none was matched.
func (e *Engine) Match(
	variableValues [][]gqlparse.Token,
	queryType gqlscan.Token,
	selectionSet []gqlparse.Token,
) (id string) {
	e.currentMask.Reset()
	fmt.Println("MASK: ", e.currentMask.String())
	e.pathScanner.Scan(
		queryType,
		selectionSet,
		func(path []byte, i int) (stop bool) {
			m, ok := e.maskByPath[string(path)]
			if !ok {
				return true
			}
			fmt.Println("PATH: ", string(path))
			e.currentMask.SetOr(e.currentMask, m)
			fmt.Println("MASK: ", e.currentMask.String())
			return false
		},
	)
	if e.currentMask.Size() < 1 {
		return ""
	}
	e.currentMask.Visit(func(i int) (skip bool) {
		t := e.templates[i]
		match := true
		t.ConstraintChecker.VisitPaths(func(path string) (stop bool) {
			if !t.ConstraintChecker.Check(path) {
				match = false
				return true
			}
			return false
		})
		if match {
			id = t.ID
			return true
		}
		return false
	})
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
