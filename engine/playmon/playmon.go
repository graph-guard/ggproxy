package playmon

import (
	"github.com/graph-guard/ggproxy/config"
	"github.com/graph-guard/ggproxy/engine/playmon/internal/constrcheck"
	"github.com/graph-guard/ggproxy/engine/playmon/internal/pathmatch"
	"github.com/graph-guard/ggproxy/engine/playmon/internal/pathscan"
	"github.com/graph-guard/ggproxy/gqlparse"
	"github.com/graph-guard/gqlscan"
	"github.com/graph-guard/gqt/v4"
)

type arg struct {
	PathHash uint64
	Value    []gqlparse.Token
}

type Engine struct {
	parser        *gqlparse.Parser
	pathScanner   *pathscan.PathScanner
	matcher       *pathmatch.Matcher
	templates     map[string]*template
	argumentPaths map[uint64]struct{}

	structuralPaths []uint64
	varValues       map[uint64][]gqlparse.Token
	argumentsSet    []arg
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
		parser:        gqlparse.NewParser(s.Schema),
		pathScanner:   pathscan.New(128, 2048),
		templates:     make(map[string]*template, len(s.Templates)),
		argumentPaths: make(map[uint64]struct{}),

		structuralPaths: make([]uint64, 1024),
		varValues:       make(map[uint64][]gqlparse.Token),
		argumentsSet:    make([]arg, 1024),
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
		e.templates[tmpl.ID] = tmpl
		if errs := pathscan.InAST(
			tmpl.GQTOpr,
			func(path uint64, e gqt.Expression) (stop bool) {
				// Structural
				return false
			}, func(path uint64, _ gqt.Expression) (stop bool) {
				// Argument
				e.argumentPaths[path] = struct{}{}
				return false
			}, func(path uint64, _ *gqt.VariableDeclaration) (stop bool) {
				// Variable
				e.varValues[path] = nil
				return false
			},
		); errs != nil {
			panic(errs)
		}
		idCounter++
	}
	e.matcher = pathmatch.New(s)
	return e
}

func (e *Engine) reset() {
	e.structuralPaths = e.structuralPaths[:0]
	e.argumentsSet = e.argumentsSet[:0]
	for path := range e.varValues {
		e.varValues[path] = nil
	}
}

// match returns the ID of the first matching template or "" if none was matched.
func (e *Engine) match(
	variableValues [][]gqlparse.Token,
	queryType gqlscan.Token,
	selectionSet []gqlparse.Token,
	onMatch func(*config.Template) (stop bool),
) {
	e.reset()
	var mismatch bool
	e.pathScanner.InTokens(
		queryType,
		selectionSet,
		e.varValues,
		func(path uint64) (stop bool) { // Structural path
			e.structuralPaths = append(e.structuralPaths, path)
			return false
		},
		func(path uint64, i int) (stop bool) { // Argument
			if _, ok := e.argumentPaths[path]; !ok {
				mismatch = true
				return false
			}
			e.argumentsSet = append(e.argumentsSet, arg{
				PathHash: path,
				Value:    selectionSet[i+1:],
			})
			return false
		},
		func(path uint64, i int) (stop bool) { // Variable value
			e.varValues[path] = selectionSet[i:]
			return false
		},
	)
	if mismatch {
		return
	}

	e.matcher.Match(e.structuralPaths, func(t *config.Template) (stop bool) {
		tm := e.templates[t.ID]
		for i := range e.argumentsSet {
			if !tm.ConstraintChecker.Check(
				variableValues,
				e.varValues,
				e.argumentsSet[i].PathHash,
				e.argumentsSet[i].Value,
			) {
				return false
			}
		}
		return onMatch(t)
	})
}

// Match calls onMatch for every matched template until onMatch returns true.
// onErr is invoked in case of an error.
func (e *Engine) Match(
	query, operationName, variablesJSON []byte,
	onParsed func(operation, selectionSet []gqlparse.Token) (stop bool),
	onMatch func(template *config.Template) (stop bool),
	onErr func(err error),
) {
	e.parser.Parse(
		query, operationName, variablesJSON,
		func(
			varVals [][]gqlparse.Token,
			operation, selectionSet []gqlparse.Token,
		) {
			if onParsed(operation, selectionSet) {
				return
			}
			e.match(
				varVals, operation[0].ID, selectionSet,
				func(t *config.Template) (stop bool) {
					return onMatch(t)
				},
			)
		},
		onErr,
	)
}
