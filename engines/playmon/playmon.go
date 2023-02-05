package playmon

import (
	"fmt"

	"github.com/graph-guard/ggproxy/engines/playmon/internal/constrcheck"
	"github.com/graph-guard/ggproxy/gqlparse"
	"github.com/graph-guard/ggproxy/utilities/bitmask"
	"github.com/graph-guard/gqlscan"
	"github.com/graph-guard/gqt/v4"
)

type Engine struct {
	constrchecker   *constrcheck.Checker
	currentMask     *bitmask.Set
	maskByPath      map[string]*bitmask.Set
	templatesByName map[string]*Template
}

type Template struct {
	Name                 string
	RawGQTTemplateSource []byte
	GQTOpr               *gqt.Operation
	ConstraintChecker    *constrcheck.Checker
}

// New expects templates to be initialized and valid.
func New(templates ...*Template) *Engine {
	e := &Engine{
		maskByPath:      make(map[string]*bitmask.Set),
		templatesByName: make(map[string]*Template, len(templates)),
	}
	for i := range templates {
		n := templates[i].Name
		if _, ok := e.templatesByName[n]; ok {
			panic(fmt.Errorf("redeclared template: %q", n))
		}
		if templates[i].GQTOpr == nil {
			panic(fmt.Errorf("uninitialized template, "+
				"missing GQT operation for path %q", n))
		}
		if templates[i].ConstraintChecker == nil {
			panic(fmt.Errorf("uninitialized template, "+
				"missing constraint checker for path %q", n))
		}
		e.templatesByName[n] = templates[i]
	}
	return e
}

// Match returns the ID of the first matching template or "" if none was matched.
func (e *Engine) Match(
	variableValues [][]gqlparse.Token,
	queryType gqlscan.Token,
	selectionSet []gqlparse.Token,
) (id string) {
	panic("todo")
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
