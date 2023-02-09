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

	"github.com/graph-guard/ggproxy/gqlparse"
	"github.com/graph-guard/ggproxy/utilities/stack"
	gqlscan "github.com/graph-guard/gqlscan"
	gqt "github.com/graph-guard/gqt/v4"
)

// PathScanner is reset in every call to InTokens
type PathScanner struct {
	pathBuf []byte
	stack   stack.Stack[int]
}

func New(preallocateStack, preallocatePathBuffer int) *PathScanner {
	return &PathScanner{
		pathBuf: make([]byte, 0, preallocatePathBuffer),
		stack:   stack.New[int](preallocateStack),
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
// and onVariable for every encountered variable.
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
	gqtVarPaths map[string]struct{},
	onStructural, onVariable func(path []byte, i int) (stop bool),
) {
	s.pathBuf = s.pathBuf[:0]
	s.stack.Reset()

	switch operation {
	case gqlscan.TokenDefQry:
		s.stackPushByte(initQuery)
	case gqlscan.TokenDefMut:
		s.stackPushByte(initMutation)
	case gqlscan.TokenDefSub:
		s.stackPushByte(initSubscription)
	default:
		panic(fmt.Errorf("unexpected operation: %v", operation))
	}
	for i, level := 0, 1; i < len(tokens); i++ {
		switch tokens[i].ID {
		case gqlscan.TokenSet:
			level++
		case gqlscan.TokenSetEnd:
			level--
			s.stackPop()
		case gqlscan.TokenFragInline:
			if level > s.stack.Len() {
				s.stackPushWithDiv(divTypeCond, tokens[i].Value)
				break
			}
			s.stackPop()
			s.stackPushWithDiv(divTypeCond, tokens[i].Value)
		case gqlscan.TokenField:
			if level > s.stack.Len() {
				s.stackPushWithDiv(divSel, tokens[i].Value)
			} else {
				s.stackPop()
				s.stackPushWithDiv(divSel, tokens[i].Value)
			}
			if t := tokens[i+1].ID; t == gqlscan.TokenArgList ||
				t == gqlscan.TokenSet {
				continue
			}
			if onStructural(s.pathBuf, i) {
				return
			}
		case gqlscan.TokenArgName:
			s.stackPushWithDiv(divArgList, tokens[i].Value)
			if onStructural(s.pathBuf, i) {
				return
			}
			s.stackPop()
		}
	}
}

func (s *PathScanner) stackPushByte(element byte) {
	s.stack.Push(1)
	s.pathBuf = append(s.pathBuf, element)
}

func (s *PathScanner) stackPushWithDiv(div byte, element []byte) {
	s.stack.Push(1 + len(element))
	s.pathBuf = append(s.pathBuf, div)
	s.pathBuf = append(s.pathBuf, element...)
}

func (s *PathScanner) stackPop() {
	t := s.stack.Top()
	s.pathBuf = s.pathBuf[:len(s.pathBuf)-t]
	s.stack.Pop()
}

// InAST calls onStructural for every structural path that can be used for
// (sub)matching. onVariable is called for every path to an argument or an
// object field that has a variable associated.
func InAST(
	o *gqt.Operation,
	onStructural func(path []byte, e gqt.Expression) (stop bool),
	onVariable func(path []byte, e gqt.Expression) (stop bool),
) {
	traverse(o, func(e gqt.Expression) (stop, skipChildren bool) {
		switch e := e.(type) {
		case *gqt.SelectionField:
			for _, a := range e.Arguments {
				if a.AssociatedVariable != nil {
					p := makePathVar(a)
					if onVariable(p, e) {
						return true, true
					}
				}
			}
			if len(e.Selections) > 0 {
				break
			}
			p := makePathStructural(e)
			if onStructural(p, e) {
				return true, true
			}
		case *gqt.ObjectField:
			if e.AssociatedVariable != nil {
				p := makePathVar(e)
				if onVariable(p, e) {
					return true, true
				}
			}
		}
		return false, false // Continue traversal
	})
}

// traverse returns true after BFS-traversing the entire tree under e
// calling onExpression for every discovered expression.
func traverse(
	e gqt.Expression,
	onExpression func(gqt.Expression) (stop, skipChildren bool),
) (stopped bool) {
	stack := make([]gqt.Expression, 0, 64)
	push := func(expression gqt.Expression) {
		stack = append(stack, expression)
	}

	push(e)
	for len(stack) > 0 {
		top := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		stop, skipChildren := onExpression(top)
		if stop {
			return true
		} else if skipChildren {
			continue
		}

		switch e := top.(type) {
		case *gqt.Operation:
			for _, f := range e.Selections {
				push(f)
			}
		case *gqt.SelectionInlineFrag:
			for _, f := range e.Selections {
				push(f)
			}
		case *gqt.SelectionField:
			for _, f := range e.Selections {
				push(f)
			}
			for _, a := range e.Arguments {
				push(a)
			}
		case *gqt.SelectionMax:
			for _, f := range e.Options.Selections {
				push(f)
			}
		case *gqt.Argument:
			push(e.Constraint)
		case *gqt.ConstrEquals:
			push(e.Value)
		case *gqt.ConstrNotEquals:
			push(e.Value)
		case *gqt.ConstrGreater:
			push(e.Value)
		case *gqt.ConstrGreaterOrEqual:
			push(e.Value)
		case *gqt.ConstrLess:
			push(e.Value)
		case *gqt.ConstrLessOrEqual:
			push(e.Value)
		case *gqt.ConstrLenEquals:
			push(e.Value)
		case *gqt.ConstrLenNotEquals:
			push(e.Value)
		case *gqt.ConstrLenGreater:
			push(e.Value)
		case *gqt.ConstrLenLess:
			push(e.Value)
		case *gqt.ConstrLenGreaterOrEqual:
			push(e.Value)
		case *gqt.ConstrLenLessOrEqual:
			push(e.Value)
		case *gqt.ConstrMap:
			push(e.Constraint)
		case *gqt.ExprParentheses:
			push(e.Expression)
		case *gqt.ExprEqual:
			push(e.Left)
			push(e.Right)
		case *gqt.ExprNotEqual:
			push(e.Left)
			push(e.Right)
		case *gqt.ExprLogicalNegation:
			push(e.Expression)
		case *gqt.ExprNumericNegation:
			push(e.Expression)
		case *gqt.ExprLogicalOr:
			for _, e := range e.Expressions {
				push(e)
			}
		case *gqt.ExprLogicalAnd:
			for _, e := range e.Expressions {
				push(e)
			}
		case *gqt.ExprAddition:
			push(e.AddendLeft)
			push(e.AddendRight)
		case *gqt.ExprSubtraction:
			push(e.Minuend)
			push(e.Subtrahend)
		case *gqt.ExprMultiplication:
			push(e.Multiplicant)
			push(e.Multiplicator)
		case *gqt.ExprDivision:
			push(e.Dividend)
			push(e.Divisor)
		case *gqt.ExprModulo:
			push(e.Dividend)
			push(e.Divisor)
		case *gqt.ExprGreater:
			push(e.Left)
			push(e.Right)
		case *gqt.ExprGreaterOrEqual:
			push(e.Left)
			push(e.Right)
		case *gqt.ExprLess:
			push(e.Left)
			push(e.Right)
		case *gqt.ExprLessOrEqual:
			push(e.Left)
			push(e.Right)
		case *gqt.Array:
			for _, i := range e.Items {
				push(i)
			}
		case *gqt.Object:
			for _, f := range e.Fields {
				push(f)
			}
		case *gqt.ObjectField:
			push(e.Constraint)
		case *gqt.Number, *gqt.True, *gqt.False, *gqt.Null,
			*gqt.Enum, *gqt.String, *gqt.ConstrAny, *gqt.Variable:
		default:
			panic(fmt.Errorf("unhandled type: %T", top))
		}
	}
	return false
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
					if i+1 < len(argNames) {
						_ = s.WriteByte(divArg)
					}
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
