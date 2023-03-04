// Package pathscan provides functions for extraction of paths
// from token slices and GQT ASTs. Paths address structural leaf nodes and
// variable values in a GraphQL operation.
// Query operations always begin with "Q", mutation operations always begin
// with "M" and subscription operations always begin with "S".
//
// Consider the following example:
//
//	query {
//		foo {
//			bar {
//				burr(x: != $i2)
//			}
//			baz {
//				buzz(b:5, a:null, c=$c:true)
//				... on Kraz {
//					fraz
//					graz(argument:{i:$c,i2=$i2:"bar"}) {
//						lum
//					}
//				}
//			}
//		}
//		mazz
//	}
//
// The above query operation contains 5 structural leafs with
// the following paths:
//
//   - Q.foo.bar.burr|x
//   - Q.foo.baz.buzz|a,b,c
//   - Q.foo.baz&Kraz.fraz
//   - Q.foo.baz&Kraz.graz|argument.lum
//   - Q.mazz
package pathscan

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/graph-guard/ggproxy/pkg/engine/playmon/internal/travgqt"
	"github.com/graph-guard/ggproxy/pkg/gqlparse"
	"github.com/graph-guard/ggproxy/pkg/stack"
	"github.com/graph-guard/ggproxy/pkg/xxhash"
	gqlscan "github.com/graph-guard/gqlscan"
	gqt "github.com/graph-guard/gqt/v4"

	"golang.org/x/exp/slices"
)

// PathScanner is reset in every call to InTokens
type PathScanner struct {
	valPathBuf        []byte
	valStack          stack.Stack[int]
	argBuf            [][]byte
	structuralPathBuf []byte
	structuralStack   stack.Stack[int]
}

func New(preallocateStack, preallocatePathBuffer int) *PathScanner {
	return &PathScanner{
		structuralPathBuf: make([]byte, 0, preallocatePathBuffer),
		structuralStack:   stack.New[int](preallocateStack),
	}
}

// Magic identifier and divider bytes
const (
	initQuery        = 'Q'
	initMutation     = 'M'
	initSubscription = 'S'
	divSel           = '.'
	divArgList       = '|'
	divArg           = ','
	divTypeCond      = '&'
	divObjField      = '/'
)

// InTokens calls onStructural for every encountered structural path,
// onArg for every argument and onGQTVarVal for every encountered
// GQT variable value.
// operation is expected to be an operation initialization token and
// is used to determine whether the path should start with
// "Q" (query), "M" (mutation) or "S" (subscription).
// InTokens will panic if operation contains any other token.
// gqtVarPaths provides a set of all known GQT variable paths.
//
// WARNING: Aliasing provided paths and using them after
// onStructural or onVariable return may cause data corruption
// because path refers to an internal buffer of PathScanner!
func (s *PathScanner) InTokens(
	operation gqlscan.Token,
	tokens []gqlparse.Token,
	gqtVarPaths map[uint64][]gqlparse.Token,
	onStructural func(pathHash uint64) (stop bool),
	onArg, onGQTVarVal func(pathHash uint64, i int) (stop bool),
) {
	s.valPathBuf = s.valPathBuf[:0]
	s.valStack.Reset()
	s.structuralPathBuf = s.structuralPathBuf[:0]
	s.structuralStack.Reset()

	switch operation {
	case gqlscan.TokenDefQry:
		s.valPathBuf = append(s.valPathBuf, initQuery)
		s.structuralPathBuf = append(s.structuralPathBuf, initQuery)
	case gqlscan.TokenDefMut:
		s.valPathBuf = append(s.valPathBuf, initMutation)
		s.structuralPathBuf = append(s.structuralPathBuf, initMutation)
	case gqlscan.TokenDefSub:
		s.valPathBuf = append(s.valPathBuf, initSubscription)
		s.structuralPathBuf = append(s.structuralPathBuf, initSubscription)
	default:
		panic(fmt.Errorf("unexpected operation: %v", operation))
	}
	for i, level := 0, 0; i < len(tokens); i++ {
		switch tokens[i].ID {
		case gqlscan.TokenSet:
			level++
		case gqlscan.TokenSetEnd:
			level--
			s.structuralPop()
			s.valPop()
		case gqlscan.TokenFragInline:
			if level <= s.structuralStack.Len() {
				s.structuralPop()
				s.valPop()
			}
			s.structuralWithDiv(divTypeCond, tokens[i].Value)
			s.valWithDiv(divTypeCond, tokens[i].Value)
		case gqlscan.TokenField:
			if level <= s.structuralStack.Len() {
				s.structuralPop()
				s.valPop()
			}
			s.valWithDiv(divSel, tokens[i].Value)
			s.structuralPathBuf = append(s.structuralPathBuf, divSel)
			s.structuralPathBuf = append(s.structuralPathBuf, tokens[i].Value...)
			l := len(tokens[i].Value) + 1

			switch tokens[i+1].ID {
			case gqlscan.TokenArgList:
				s.argBuf = s.argBuf[:0]
				// Collect and sort arguments
				level++
				for i += 2; tokens[i].ID != gqlscan.TokenArgListEnd; i++ {
					switch tokens[i].ID {
					case gqlscan.TokenArgName:
						if level <= s.valStack.Len() {
							s.valPop()
						}
						s.argBuf = append(s.argBuf, tokens[i].Value)
						s.valWithDiv(divArgList, tokens[i].Value)
						h := Hash(s.valPathBuf)
						onArg(h, i)
						if _, ok := gqtVarPaths[h]; ok {
							if onGQTVarVal(h, i+1) {
								return
							}
						}
					case gqlscan.TokenObj:
						if tokens[i+1].ID == gqlscan.TokenObjEnd {
							// Empty object
							i++
							break
						}
						level++
					case gqlscan.TokenObjEnd:
						level--
						s.valPop()
					case gqlscan.TokenObjField:
						if level <= s.valStack.Len() {
							s.valPop()
						}
						s.valWithDiv(divObjField, tokens[i].Value)
						h := Hash(s.valPathBuf)
						if _, ok := gqtVarPaths[h]; ok {
							if onGQTVarVal(h, i+1) {
								return
							}
						}
					}
				}
				level--
				s.valPop() // Pop last argument
				slices.SortFunc(s.argBuf, func(i, j []byte) bool {
					return bytes.Compare(i, j) < 0
				})

				// Write args to path
				s.structuralPathBuf = append(s.structuralPathBuf, divArgList)
				l++
				for i := range s.argBuf {
					s.structuralPathBuf = append(s.structuralPathBuf, s.argBuf[i]...)
					s.structuralPathBuf = append(s.structuralPathBuf, divArg)
					l += len(s.argBuf[i]) + 1
				}
				s.structuralStack.Push(l)

				// Check for leaf
				if tokens[i+1].ID != gqlscan.TokenSet {
					if onStructural(Hash(s.structuralPathBuf)) {
						return
					}
					s.structuralPop()
					s.valPop()
					continue
				}
			case gqlscan.TokenSet:
				s.structuralStack.Push(l)
			default:
				s.structuralStack.Push(l)
				if onStructural(Hash(s.structuralPathBuf)) {
					return
				}
				s.structuralPop()
				s.valPop()
			}
		}
	}
}

func (s *PathScanner) valWithDiv(div byte, element []byte) {
	s.valStack.Push(1 + len(element))
	s.valPathBuf = append(s.valPathBuf, div)
	s.valPathBuf = append(s.valPathBuf, element...)
}

func (s *PathScanner) valPop() {
	t := s.valStack.Top()
	s.valPathBuf = s.valPathBuf[:len(s.valPathBuf)-t]
	s.valStack.Pop()
}

func (s *PathScanner) structuralWithDiv(div byte, element []byte) {
	s.structuralStack.Push(1 + len(element))
	s.structuralPathBuf = append(s.structuralPathBuf, div)
	s.structuralPathBuf = append(s.structuralPathBuf, element...)
}

func (s *PathScanner) structuralPop() {
	t := s.structuralStack.Top()
	s.structuralPathBuf = s.structuralPathBuf[:len(s.structuralPathBuf)-t]
	s.structuralStack.Pop()
}

// InAST calls onStructural for every structural path that can be used for
// (sub)matching. onVariable is called for every path to an argument or an
// object field that has a variable associated. onArg is called for every
// argument.
func InAST(
	o *gqt.Operation,
	onStructural func(pathHash uint64, e gqt.Expression) (stop bool),
	onArg func(pathHash uint64, e gqt.Expression) (stop bool),
	onVariable func(pathHash uint64, e *gqt.VariableDeclaration) (stop bool),
) (errs []error) {
	hashes := map[uint64]string{}
	hash := func(path []byte) uint64 {
		h := Hash(path)
		s := string(path)
		if x, ok := hashes[h]; ok {
			if x != s {
				errs = append(errs, fmt.Errorf(
					"hash collision between %q and %q (%d)",
					x, s, h,
				))
			}
		}
		hashes[h] = s
		return h
	}

	travgqt.Traverse(o, func(e gqt.Expression) (stop, skipChildren bool) {
		switch e := e.(type) {
		case *gqt.SelectionField:
			for _, a := range e.Arguments {
				p := makePathVar(a)
				if onArg(hash(p), a) {
					return true, true
				}
				if a.AssociatedVariable != nil {
					p := makePathVar(a)
					if onVariable(hash(p), a.AssociatedVariable) {
						return true, true
					}
				}
			}
			if len(e.Selections) > 0 {
				break
			}
			p := makePathStructural(e)
			hashes[hash(p)] = string(p)
			if onStructural(hash(p), e) {
				return true, true
			}
		case *gqt.ObjectField:
			if e.AssociatedVariable != nil {
				p := makePathVar(e)
				if onVariable(hash(p), e.AssociatedVariable) {
					return true, true
				}
			}
		}
		return false, false // Continue traversal
	})
	return errs
}

// makePathStructural generates a structural path for the given expression.
func makePathStructural(e *gqt.SelectionField) []byte {
	var p []gqt.Expression // Reversed path
	for e := gqt.Expression(e); e != nil; e = e.GetParent() {
		p = append(p, e)
	}
	var s bytes.Buffer
	for i := len(p) - 1; i >= 0; i-- {
		switch v := p[i].(type) {
		case *gqt.SelectionInlineFrag:
			_ = s.WriteByte('&')
			_, _ = s.WriteString(v.TypeCondition.TypeName)
		case *gqt.SelectionField:
			_ = s.WriteByte('.')
			_, _ = s.WriteString(v.Name.Name)
			if len(v.Arguments) > 0 {
				_ = s.WriteByte('|')
				argNames := make([]string, len(v.Arguments))
				for i := range v.Arguments {
					argNames[i] = v.Arguments[i].Name.Name
				}
				sort.Strings(argNames)
				for i := range argNames {
					_, _ = s.WriteString(argNames[i])
					_ = s.WriteByte(divArg)
				}
			}
		case *gqt.Operation:
			switch v.Type {
			case gqt.OperationTypeQuery:
				_, _ = s.WriteString("Q")
			case gqt.OperationTypeMutation:
				_, _ = s.WriteString("M")
			case gqt.OperationTypeSubscription:
				_, _ = s.WriteString("S")
			default:
				panic(fmt.Errorf("unknown operation type: %d", v.Type))
			}
		}
	}
	return s.Bytes()
}

func makePathVar(e gqt.Expression) []byte {
	var p []gqt.Expression // Reversed path
	for e := e; e != nil; e = e.GetParent() {
		p = append(p, e)
	}
	var s bytes.Buffer
	for i := len(p) - 1; i >= 0; i-- {
		switch v := p[i].(type) {
		case *gqt.SelectionInlineFrag:
			_ = s.WriteByte(divTypeCond)
			_, _ = s.WriteString(v.TypeCondition.TypeName)
		case *gqt.SelectionField:
			_ = s.WriteByte(divSel)
			_, _ = s.WriteString(v.Name.Name)
		case *gqt.Argument:
			_ = s.WriteByte(divArgList)
			_, _ = s.WriteString(v.Name.Name)
		case *gqt.ObjectField:
			_ = s.WriteByte(divObjField)
			_, _ = s.WriteString(v.Name.Name)
		case *gqt.Operation:
			switch v.Type {
			case gqt.OperationTypeQuery:
				_ = s.WriteByte(initQuery)
			case gqt.OperationTypeMutation:
				_ = s.WriteByte(initMutation)
			case gqt.OperationTypeSubscription:
				_ = s.WriteByte(initSubscription)
			default:
				panic(fmt.Errorf("unknown operation type: %d", v.Type))
			}
		}
	}
	return s.Bytes()
}

func Hash[B []byte | string](b B) uint64 {
	h := xxhash.New(0)
	xxhash.Write(&h, b)
	return h.Sum64()
}
