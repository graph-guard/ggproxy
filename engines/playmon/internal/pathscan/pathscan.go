// Package pathscan provides a scanner that traverses token slices and
// finds all paths of leaf nodes in a GraphQL operation.
// Query operations always begin with "Q", mutation operations always begin
// with "M" and subscription operations always begin with "S".
//
// Consider the following example:
//
//	query {
//		foo {
//			bar {
//				burr(x:4)
//			}
//			baz {
//				buzz(b:5, a:null, c:true)
//				... on Kraz {
//					fraz
//					graz(argument:{i:"foo",i2:"bar"}) {
//						lum
//					}
//				}
//			}
//		}
//		mazz
//	}
//
// The above query contains 7 leafs with the following paths:
//
//   - Q.foo.bar.burr|x
//   - Q.foo.baz.buzz|b
//   - Q.foo.baz.buzz|a
//   - Q.foo.baz.buzz|c
//   - Q.foo.baz&Kraz.fraz
//   - Q.foo.baz&Kraz.graz|argument
//   - Q.foo.baz&Kraz.graz.lum
//   - Q.mazz
package pathscan

import (
	"github.com/graph-guard/ggproxy/gqlparse"
	"github.com/graph-guard/ggproxy/utilities/stack"
	gqlscan "github.com/graph-guard/gqlscan"
)

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
	elemQuery        = 'Q'
	elemMutation     = 'M'
	elemSubscription = 'S'
	divField         = '.'
	divArg           = '|'
	divTypeCond      = '&'
)

func (s *PathScanner) Scan(tokens []gqlparse.Token, onPath func([]byte) (stop bool)) {
	s.pathBuf = s.pathBuf[:0]
	s.stack.Reset()
	for i, level := 0, 1; i < len(tokens); i++ {
		switch tokens[i].ID {
		case gqlscan.TokenDefQry:
			s.stackPushByte(elemQuery)
		case gqlscan.TokenDefMut:
			s.stackPushByte(elemMutation)
		case gqlscan.TokenDefSub:
			s.stackPushByte(elemSubscription)
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
				s.stackPushWithDiv(divField, tokens[i].Value)
			} else {
				s.stackPop()
				s.stackPushWithDiv(divField, tokens[i].Value)
			}
			if t := tokens[i+1].ID; t == gqlscan.TokenArgList ||
				t == gqlscan.TokenSet {
				continue
			}
			if onPath(s.pathBuf) {
				return
			}
		case gqlscan.TokenArgName:
			s.stackPushWithDiv(divArg, tokens[i].Value)
			if onPath(s.pathBuf) {
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
