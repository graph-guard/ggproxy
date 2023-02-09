package constrcheck

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/graph-guard/ggproxy/engine/playmon/internal/constrcheck/internal/union"
	"github.com/graph-guard/ggproxy/gqlparse"
	"github.com/graph-guard/ggproxy/utilities/atoi"
	"github.com/graph-guard/gqlscan"
	"github.com/graph-guard/gqt/v4"
	gqlast "github.com/vektah/gqlparser/v2/ast"
)

// Enum is a GraphQL enum value.
type Enum string

// Checker is a constraint checker instance.
// Before calling Check, make sure you initialize the Checker using Init.
type Checker struct {
	operation *gqt.Operation
	schema    *gqlast.Schema
	checkers  map[string]check

	variableValues map[string][]gqlparse.Token

	inputValue   []gqlparse.Token
	checkedValue []gqlparse.Token
	stack        []union.Union
}

// // Paths executes fn for every path the checker is aware of.
// // Returns true immediatelly if any call to fn returns true.
// func (m *Checker) Paths(fn func(string) (stop bool)) (stopped bool) {
// 	for p := range m.checkers {
// 		if fn(p) {
// 			return true
// 		}
// 	}
// 	return false
// }

func (m *Checker) stackTopType() union.Type {
	return m.stack[len(m.stack)-1].Type()
}

func (m *Checker) popStack() union.Union {
	t := m.stack[len(m.stack)-1]
	m.stack = m.stack[:len(m.stack)-1]
	return t
}

func (m *Checker) popStackArrays() (left, right []union.Union) {
	if m.stack[len(m.stack)-1].Type() != union.TypeArray {
		return
	}
	i := len(m.stack) - 2
	var c int
LOOP_R:
	for l := 1; i > 0; i, c = i-1, c+1 {
		switch m.stack[i].Type() {
		case union.TypeArray:
			l++
		case union.TypeArrayEnd:
			l--
			if l < 1 {
				break LOOP_R
			}
		}
	}
	right = m.stack[i+1 : len(m.stack)-1]

	i--
	iv := i
	if m.stack[i].Type() != union.TypeArray {
		return
	}
	c = 0
	i--
LOOP_L:
	for l := 1; i > 0; i, c = i-1, c+1 {
		switch m.stack[i].Type() {
		case union.TypeArray:
			l++
		case union.TypeArrayEnd:
			l--
			if l < 1 {
				break LOOP_L
			}
		}
	}

	left = m.stack[i+1 : iv]
	m.stack = m.stack[:i]
	return left, right
}

func (m *Checker) pushStackArray() {
	m.stack = append(m.stack, union.Array())
}

func (m *Checker) pushStackArrayEnd() {
	m.stack = append(m.stack, union.ArrayEnd())
}

func (m *Checker) pushStackInt(v int32) {
	m.stack = append(m.stack, union.Int(v))
}

func (m *Checker) pushStackFloat(v float64) {
	m.stack = append(m.stack, union.Float(v))
}

func (m *Checker) pushStackBool(v bool) {
	if v {
		m.stack = append(m.stack, union.True())
		return
	}
	m.stack = append(m.stack, union.False())
}

// Init initializes the checker with inputs to validate a request.
// Init must be used before making calls to Check in the context of a request.
func (m *Checker) Init(variableValues map[string][]gqlparse.Token) {
	m.variableValues = variableValues
}

// Check returns true if the value for the given path is accepted,
// otherwise returns false.
func (m *Checker) Check(path string) bool {
	m.stack = m.stack[:0]
	c := m.checkers[path]
	if c == nil {
		return false
	}
	v := m.variableValues[path]
	m.inputValue, m.checkedValue = v, v
	return c(m)
}

// check is a constraint check function which returns true
// if the input value matches the constraint and can be accepted,
// otherwise returns false.
type check func(*Checker) (match bool)

// resolveExpr resolves expression e into a union and pushes it onto the stack.
func (m *Checker) resolveExpr(e gqt.Expression) union.Type {
	switch e := e.(type) {
	case *gqt.Array:
		m.pushStackArrayEnd()
		for i := len(e.Items) - 1; i >= 0; i-- {
			m.resolveExpr(e.Items[i].(*gqt.ConstrEquals).Value)
		}
		m.pushStackArray()
		return union.TypeArray
	case *gqt.Variable:
		p := MakePath(e.Declaration.Parent)
		if s := m.variableValues[p]; s != nil {
			m.stack = append(m.stack, union.Tokens(s))
			return union.TypeTokens
		}
		m.stack = append(m.stack, union.Null())
		return union.TypeNull
	case *gqt.Number:
		if i, ok := e.Int(); ok {
			m.stack = append(m.stack, union.Int(int32(i)))
			return union.TypeInt
		}
		f, _ := e.Float()
		m.stack = append(m.stack, union.Float(f))
		return union.TypeFloat
	case *gqt.True:
		m.stack = append(m.stack, union.True())
		return union.TypeBoolean
	case *gqt.False:
		m.stack = append(m.stack, union.False())
		return union.TypeBoolean
	case *gqt.Enum:
		m.stack = append(m.stack, union.Enum(e.Value))
		return union.TypeEnum
	case *gqt.String:
		m.stack = append(m.stack, union.String(e.Value))
		return union.TypeString
	case *gqt.Null:
		return union.TypeNull
	case *gqt.ExprAddition:
		m.resolveExpr(e.AddendLeft)
		m.resolveExpr(e.AddendRight)
		r := m.popStack()
		l := m.popStack()
		if l.Type() == union.TypeInt && r.Type() == union.TypeInt {
			l, _ := l.Int()
			r, _ := r.Int()
			m.pushStackInt(l + r)
			return union.TypeInt
		}
		{
			l, _ := l.Float()
			r, _ := r.Float()
			m.pushStackFloat(l + r)
		}
		return union.TypeFloat
	case *gqt.ExprSubtraction:
		m.resolveExpr(e.Minuend)
		m.resolveExpr(e.Subtrahend)
		r := m.popStack()
		l := m.popStack()
		if l.Type() == union.TypeInt && r.Type() == union.TypeInt {
			l, _ := l.Int()
			r, _ := r.Int()
			m.pushStackInt(l - r)
			return union.TypeInt
		}
		{
			l, _ := l.Float()
			r, _ := r.Float()
			m.pushStackFloat(l - r)
		}
		return union.TypeFloat
	case *gqt.ExprMultiplication:
		m.resolveExpr(e.Multiplicant)
		m.resolveExpr(e.Multiplicator)
		r := m.popStack()
		l := m.popStack()
		if l.Type() == union.TypeInt && r.Type() == union.TypeInt {
			l, _ := l.Int()
			r, _ := r.Int()
			m.pushStackInt(l * r)
			return union.TypeInt
		}
		{
			l, _ := l.Float()
			r, _ := r.Float()
			m.pushStackFloat(l * r)
		}
		return union.TypeFloat
	case *gqt.ExprDivision:
		m.resolveExpr(e.Dividend)
		m.resolveExpr(e.Divisor)
		r := m.popStack()
		l := m.popStack()
		if l.Type() == union.TypeInt && r.Type() == union.TypeInt {
			l, _ := l.Int()
			r, _ := r.Int()
			m.pushStackInt(l / r)
			return union.TypeInt
		}
		{
			l, _ := l.Float()
			r, _ := r.Float()
			m.pushStackFloat(l / r)
		}
		return union.TypeFloat
	case *gqt.ExprModulo:
		m.resolveExpr(e.Dividend)
		m.resolveExpr(e.Divisor)
		r := m.popStack()
		l := m.popStack()
		if l.Type() == union.TypeInt && r.Type() == union.TypeInt {
			l, _ := l.Int()
			r, _ := r.Int()
			m.pushStackInt(l % r)
			return union.TypeInt
		}
		{
			l, _ := l.Float()
			r, _ := r.Float()
			m.pushStackFloat(math.Mod(l, r))
		}
		return union.TypeFloat
	case *gqt.ExprNumericNegation:
		m.resolveExpr(e.Expression)
		i := m.popStack()
		switch i.Type() {
		case union.TypeFloat:
			i, _ := i.Float()
			m.pushStackFloat(-i)
			return union.TypeFloat
		case union.TypeInt:
			i, _ := i.Int()
			m.pushStackInt(-i)
			return union.TypeInt
		case union.TypeTokens:
			switch i.Tokens()[0].ID {
			case gqlscan.TokenInt:
				i := atoi.MustI32(i.Tokens()[0].Value)
				m.pushStackInt(-i)
				return union.TypeInt
			case gqlscan.TokenFloat:
				f := atoi.MustF64(i.Tokens()[0].Value)
				m.pushStackFloat(-f)
				return union.TypeFloat
			}
			panic(fmt.Errorf("unexpected token type: %q", i.Tokens()[0].ID.String()))
		}
		panic(fmt.Errorf("unexpected union type: %q", i.Type().String()))
	case *gqt.ExprEqual:
		m.resolveExpr(e.Left)
		m.resolveExpr(e.Right)
		equal := false
		if m.stackTopType() == union.TypeArray {
			left, right := m.popStackArrays()
			equal = unionsEqual(left, right)
		} else {
			r := m.popStack()
			l := m.popStack()
			equal = union.Equal(l, r)
		}
		m.pushStackBool(equal)
		return union.TypeBoolean
	case *gqt.ExprNotEqual:
		m.resolveExpr(e.Left)
		m.resolveExpr(e.Right)
		notEqual := false
		if m.stackTopType() == union.TypeArray {
			left, right := m.popStackArrays()
			notEqual = !unionsEqual(left, right)
		} else {
			r := m.popStack()
			l := m.popStack()
			notEqual = !union.Equal(l, r)
		}
		m.pushStackBool(notEqual)
		return union.TypeBoolean
	case *gqt.ExprLogicalNegation:
		m.resolveExpr(e.Expression)
		u := m.popStack()
		b, _ := u.Bool()
		m.pushStackBool(!b)
		return union.TypeBoolean
	case *gqt.ExprGreater:
		m.resolveExpr(e.Left)
		m.resolveExpr(e.Right)
		r := m.popStack()
		l := m.popStack()
		if l.Type() == union.TypeInt && r.Type() == union.TypeInt {
			l, _ := l.Int()
			r, _ := r.Int()
			m.pushStackBool(l > r)
		} else {
			l, _ := l.Float()
			r, _ := r.Float()
			m.pushStackBool(l > r)
		}
		return union.TypeBoolean
	case *gqt.ExprLess:
		m.resolveExpr(e.Left)
		m.resolveExpr(e.Right)
		r := m.popStack()
		l := m.popStack()
		if l.Type() == union.TypeInt && r.Type() == union.TypeInt {
			l, _ := l.Int()
			r, _ := r.Int()
			m.pushStackBool(l < r)
		} else {
			l, _ := l.Float()
			r, _ := r.Float()
			m.pushStackBool(l < r)
		}
		return union.TypeBoolean
	case *gqt.ExprGreaterOrEqual:
		m.resolveExpr(e.Left)
		m.resolveExpr(e.Right)
		r := m.popStack()
		l := m.popStack()
		if l.Type() == union.TypeInt && r.Type() == union.TypeInt {
			l, _ := l.Int()
			r, _ := r.Int()
			m.pushStackBool(l >= r)
		} else {
			l, _ := l.Float()
			r, _ := r.Float()
			m.pushStackBool(l >= r)
		}
		return union.TypeBoolean
	case *gqt.ExprLessOrEqual:
		m.resolveExpr(e.Left)
		m.resolveExpr(e.Right)
		r := m.popStack()
		l := m.popStack()
		if l.Type() == union.TypeInt && r.Type() == union.TypeInt {
			l, _ := l.Int()
			r, _ := r.Int()
			m.pushStackBool(l <= r)
		} else {
			l, _ := l.Float()
			r, _ := r.Float()
			m.pushStackBool(l <= r)
		}
		return union.TypeBoolean
	case *gqt.ExprLogicalOr:
		for _, x := range e.Expressions {
			m.resolveExpr(x)
			u := m.popStack()
			b, _ := u.Bool()
			if b {
				m.pushStackBool(true)
				return union.TypeBoolean
			}
		}
		m.pushStackBool(false)
		return union.TypeBoolean
	case *gqt.ExprLogicalAnd:
		for _, x := range e.Expressions {
			m.resolveExpr(x)
			u := m.popStack()
			b, _ := u.Bool()
			if !b {
				m.pushStackBool(false)
				return union.TypeBoolean
			}
		}
		m.pushStackBool(true)
		return union.TypeBoolean
	case *gqt.ExprParentheses:
		return m.resolveExpr(e.Expression)
	}
	panic(fmt.Errorf("unhandled value expression type: %T", e))
}

func (m *Checker) PathsLen() int { return len(m.checkers) }

func (m *Checker) VisitPaths(fn func(path string) (stop bool)) {
	for path := range m.checkers {
		if fn(path) {
			return
		}
	}
}

func (m *Checker) VisitPathsAll(fn func(path string)) {
	for path := range m.checkers {
		fn(path)
	}
}

// New creates a constraint checker instance for each path of o.
func New(o *gqt.Operation, s *gqlast.Schema) *Checker {
	m := &Checker{
		operation: o,
		schema:    s,
		checkers:  make(map[string]check),
		stack:     make([]union.Union, 1024),
	}

	inputTypes := map[string]*gqlast.Type{}
	if s != nil {
		traverse(o, func(e gqt.Expression) (stop, skipChildren bool) {
			switch e := e.(type) {
			case *gqt.SelectionField:
				for _, a := range e.Arguments {
					if a.Def == nil {
						continue
					}
					p := MakePath(a)
					inputTypes[p] = a.Def.Type
				}
			case *gqt.ObjectField:
				if e.Def == nil {
					break
				}
				p := MakePath(e)
				inputTypes[p] = e.Def.Type
			}
			return false, false
		})
	}

	// TODO: move path scanner out of this package
	traverse(o, func(e gqt.Expression) (stop, skipChildren bool) {
		switch e := e.(type) {
		case *gqt.SelectionField:
			if len(e.Arguments) < 1 && len(e.Selections) < 1 {
				p := MakePath(e)
				m.checkers[p] = nil
				break
			}
			for _, a := range e.Arguments {
				var expect *gqlast.Type
				if a.Def != nil {
					expect = a.Def.Type
				}
				if fn := makeCheck(a.Constraint, expect, s); fn != nil {
					p := MakePath(a)
					m.checkers[p] = fn
				}
			}
		}
		return false, false // Continue traversal
	})

	return m
}

func makeCheck(
	e gqt.Expression,
	expect *gqlast.Type,
	schema *gqlast.Schema,
) check {
	switch e := e.(type) {
	case *gqt.ConstrAny:
		return func(m *Checker) (match bool) {
			if wrong, _ := isWrongType(expect, m.checkedValue, schema); wrong {
				return false
			}

			// Make sure the value is semantically valid.
			return !gqlparse.ScanValuesInArrays(m.checkedValue, func(t []gqlparse.Token) (stop bool) {
				if t[0].ID != gqlscan.TokenObj {
					return false
				}
				t = t[1:]
				checkedFields := map[string]struct{}{}
				for t[0].ID != gqlscan.TokenObjEnd {
					fieldName := t[0].Value
					if _, ok := checkedFields[string(fieldName)]; ok {
						// Duplicate field! Invalid object value.
						return true
					}
					checkedFields[string(fieldName)] = struct{}{}
					t = t[1:]
				}
				return false
			})
		}
	case *gqt.ExprParentheses:
		return makeCheck(e.Expression, expect, schema)
	case *gqt.ExprLogicalOr:
		exprCheckers := make([]check, len(e.Expressions))
		for i, e := range e.Expressions {
			exprCheckers[i] = makeCheck(e, expect, schema)
		}
		return func(m *Checker) (match bool) {
			r := m.checkedValue
			for _, e := range exprCheckers {
				if e(m) {
					return true
				}
				// Reset value to start checking from the same index
				m.checkedValue = r
			}
			return false
		}
	case *gqt.ExprLogicalAnd:
		exprCheckers := make([]check, len(e.Expressions))
		for i, e := range e.Expressions {
			exprCheckers[i] = makeCheck(e, expect, schema)
		}
		return func(m *Checker) (match bool) {
			r := m.checkedValue
			for _, e := range exprCheckers {
				if !e(m) {
					return false
				}
				// Reset value to start checking from the same index
				m.checkedValue = r
			}
			return true
		}
	case *gqt.ConstrGreater:
		return func(m *Checker) (match bool) {
			if m.expectOrNum(expect) {
				return false
			}
			m.resolveExpr(e.Value)
			u := m.popStack()
			switch m.checkedValue[0].ID {
			case gqlscan.TokenInt:
				i := atoi.MustI32(m.checkedValue[0].Value)
				if u, ok := u.Int(); ok != union.ValueNone {
					match = i > u
					break
				}
				u, _ := u.Float()
				match = float64(i) > u
			case gqlscan.TokenFloat:
				i := atoi.MustF64(m.checkedValue[0].Value)
				u, _ := u.Float()
				match = i > u
			}
			m.checkedValue = m.checkedValue[1:]
			return match
		}
	case *gqt.ConstrGreaterOrEqual:
		return func(m *Checker) (match bool) {
			if m.expectOrNum(expect) {
				return false
			}
			m.resolveExpr(e.Value)
			u := m.popStack()
			switch m.checkedValue[0].ID {
			case gqlscan.TokenInt:
				i := atoi.MustI32(m.checkedValue[0].Value)
				if u, ok := u.Int(); ok != union.ValueNone {
					match = i >= u
					break
				}
				u, _ := u.Float()
				match = float64(i) >= u
			case gqlscan.TokenFloat:
				i := atoi.MustF64(m.checkedValue[0].Value)
				u, _ := u.Float()
				match = i >= u
			}
			m.checkedValue = m.checkedValue[1:]
			return match
		}
	case *gqt.ConstrLess:
		return func(m *Checker) (match bool) {
			if m.expectOrNum(expect) {
				return false
			}
			m.resolveExpr(e.Value)
			u := m.popStack()
			switch m.checkedValue[0].ID {
			case gqlscan.TokenInt:
				i := atoi.MustI32(m.checkedValue[0].Value)
				if u, ok := u.Int(); ok != union.ValueNone {
					match = i < u
					break
				}
				u, _ := u.Float()
				match = float64(i) < u
			case gqlscan.TokenFloat:
				i := atoi.MustF64(m.checkedValue[0].Value)
				u, _ := u.Float()
				match = i < u
			}
			m.checkedValue = m.checkedValue[1:]
			return match
		}
	case *gqt.ConstrLessOrEqual:
		return func(m *Checker) (match bool) {
			if m.expectOrNum(expect) {
				return false
			}
			m.resolveExpr(e.Value)
			u := m.popStack()
			switch m.checkedValue[0].ID {
			case gqlscan.TokenInt:
				i := atoi.MustI32(m.checkedValue[0].Value)
				if u, ok := u.Int(); ok != union.ValueNone {
					match = i <= u
					break
				}
				u, _ := u.Float()
				match = float64(i) <= u
			case gqlscan.TokenFloat:
				i := atoi.MustF64(m.checkedValue[0].Value)
				u, _ := u.Float()
				match = i <= u
			}
			m.checkedValue = m.checkedValue[1:]
			return match
		}
	case *gqt.ConstrEquals:
		switch v := e.Value.(type) {
		case *gqt.Array:
			return makeEqArray(v, expect, schema)
		case *gqt.Object:
			return makeEqObject(v, expect, schema)
		case *gqt.Number:
			return makeEqNumber(v, expect, schema)
		case *gqt.String:
			return makeEqString(v, expect, schema)
		case *gqt.Enum:
			return makeEqEnum(v, expect, schema)
		case *gqt.Null:
			return makeEqNull(expect, schema)
		case *gqt.False:
			return makeEqBool(false, expect, schema)
		case *gqt.True:
			return makeEqBool(true, expect, schema)
		}
		// Expression
		return makeEqExpression(e.Value, expect, schema)
	case *gqt.ConstrNotEquals:
		switch v := e.Value.(type) {
		case *gqt.Array:
			return makeNotEqArray(v, expect, schema)
		case *gqt.Object:
			return makeNotEqObject(v, expect, schema)
		case *gqt.Number:
			return makeNotEqNumber(v, expect, schema)
		case *gqt.String:
			return makeNotEqString(v, expect, schema)
		case *gqt.Enum:
			return makeNotEqEnum(v, expect, schema)
		case *gqt.Null:
			return makeNotEqNull(expect, schema)
		case *gqt.False:
			return makeNotEqBool(false, expect, schema)
		case *gqt.True:
			return makeNotEqBool(true, expect, schema)
		}
		fn := makeEqExpression(e.Value, expect, schema)
		return func(m *Checker) (match bool) {
			return !fn(m)
		}
	case *gqt.ConstrLenEquals:
		return func(m *Checker) (match bool) {
			if m.expectOrHasLen(expect) {
				return false
			}

			var length int
			switch m.checkedValue[0].ID {
			case gqlscan.TokenArr:
				var tokensRead int
				length, tokensRead = gqlparse.CountValuesUntil(
					m.checkedValue[1:],
					gqlscan.TokenArrEnd,
				)
				m.checkedValue = m.checkedValue[tokensRead:]
			case gqlscan.TokenStr:
				length = len(m.checkedValue[0].Value)
			case gqlscan.TokenStrBlock:
				length = len(m.checkedValue[0].Value)
			}

			m.resolveExpr(e.Value)
			u := m.popStack()
			if u, ok := u.Int(); ok != union.ValueNone {
				return int32(length) == u
			}
			if u, ok := u.Float(); ok != union.ValueNone {
				return float64(length) == u
			}
			panic(fmt.Errorf("unexpected value type: %s", u.Type().String()))
		}
	case *gqt.ConstrLenNotEquals:
		return func(m *Checker) (match bool) {
			if m.expectOrHasLen(expect) {
				return false
			}

			var length int
			switch m.checkedValue[0].ID {
			case gqlscan.TokenArr:
				var tokensRead int
				length, tokensRead = gqlparse.CountValuesUntil(
					m.checkedValue[1:],
					gqlscan.TokenArrEnd,
				)
				m.checkedValue = m.checkedValue[tokensRead:]
			case gqlscan.TokenStr:
				length = len(m.checkedValue[0].Value)
			case gqlscan.TokenStrBlock:
				length = len(m.checkedValue[0].Value)
			}

			m.resolveExpr(e.Value)
			u := m.popStack()
			if u, ok := u.Int(); ok != union.ValueNone {
				return int32(length) != u
			}
			if u, ok := u.Float(); ok != union.ValueNone {
				return float64(length) != u
			}
			panic(fmt.Errorf("unexpected value type: %s", u.Type().String()))
		}
	case *gqt.ConstrLenGreater:
		return func(m *Checker) (match bool) {
			if m.expectOrHasLen(expect) {
				return false
			}

			var length int
			switch m.checkedValue[0].ID {
			case gqlscan.TokenArr:
				var tokensRead int
				length, tokensRead = gqlparse.CountValuesUntil(
					m.checkedValue[1:],
					gqlscan.TokenArrEnd,
				)
				m.checkedValue = m.checkedValue[tokensRead:]
			case gqlscan.TokenStr:
				length = len(m.checkedValue[0].Value)
			case gqlscan.TokenStrBlock:
				length = len(m.checkedValue[0].Value)
			}

			m.resolveExpr(e.Value)
			u := m.popStack()
			if u, ok := u.Int(); ok != union.ValueNone {
				return int32(length) > u
			}
			if u, ok := u.Float(); ok != union.ValueNone {
				return float64(length) > u
			}
			panic(fmt.Errorf("unexpected value type: %s", u.Type().String()))
		}
	case *gqt.ConstrLenGreaterOrEqual:
		return func(m *Checker) (match bool) {
			if m.expectOrHasLen(expect) {
				return false
			}

			var length int
			switch m.checkedValue[0].ID {
			case gqlscan.TokenArr:
				var tokensRead int
				length, tokensRead = gqlparse.CountValuesUntil(
					m.checkedValue[1:],
					gqlscan.TokenArrEnd,
				)
				m.checkedValue = m.checkedValue[tokensRead:]
			case gqlscan.TokenStr:
				length = len(m.checkedValue[0].Value)
			case gqlscan.TokenStrBlock:
				length = len(m.checkedValue[0].Value)
			}

			m.resolveExpr(e.Value)
			u := m.popStack()
			if u, ok := u.Int(); ok != union.ValueNone {
				return int32(length) >= u
			}
			if u, ok := u.Float(); ok != union.ValueNone {
				return float64(length) >= u
			}
			panic(fmt.Errorf("unexpected value type: %s", u.Type().String()))
		}
	case *gqt.ConstrLenLess:
		return func(m *Checker) (match bool) {
			if m.expectOrHasLen(expect) {
				return false
			}

			var length int
			switch m.checkedValue[0].ID {
			case gqlscan.TokenArr:
				var tokensRead int
				length, tokensRead = gqlparse.CountValuesUntil(
					m.checkedValue[1:],
					gqlscan.TokenArrEnd,
				)
				m.checkedValue = m.checkedValue[tokensRead:]
			case gqlscan.TokenStr:
				length = len(m.checkedValue[0].Value)
			case gqlscan.TokenStrBlock:
				length = len(m.checkedValue[0].Value)
			}

			m.resolveExpr(e.Value)
			u := m.popStack()
			if u, ok := u.Int(); ok != union.ValueNone {
				return int32(length) < u
			}
			if u, ok := u.Float(); ok != union.ValueNone {
				return float64(length) < u
			}
			panic(fmt.Errorf("unexpected value type: %s", u.Type().String()))
		}
	case *gqt.ConstrLenLessOrEqual:
		return func(m *Checker) (match bool) {
			if m.expectOrHasLen(expect) {
				return false
			}

			var length int
			switch m.checkedValue[0].ID {
			case gqlscan.TokenArr:
				var tokensRead int
				length, tokensRead = gqlparse.CountValuesUntil(
					m.checkedValue[1:],
					gqlscan.TokenArrEnd,
				)
				m.checkedValue = m.checkedValue[tokensRead:]
			case gqlscan.TokenStr:
				length = len(m.checkedValue[0].Value)
			case gqlscan.TokenStrBlock:
				length = len(m.checkedValue[0].Value)
			}

			m.resolveExpr(e.Value)
			u := m.popStack()
			if u, ok := u.Int(); ok != union.ValueNone {
				return int32(length) <= u
			}
			if u, ok := u.Float(); ok != union.ValueNone {
				return float64(length) <= u
			}
			panic(fmt.Errorf("unexpected value type: %s", u.Type().String()))
		}
	case *gqt.ConstrMap:
		var expectItem *gqlast.Type
		if expect != nil {
			expectItem = expect.Elem
		}
		itemCheck := makeCheck(e.Constraint, expectItem, schema)
		return func(m *Checker) (match bool) {
			if wrong, _ := isWrongType(expect, m.checkedValue, schema); wrong {
				return false
			}
			for m.checkedValue[0].ID != gqlscan.TokenArr {
				return false
			}
			m.checkedValue = m.checkedValue[1:]
			for m.checkedValue[0].ID != gqlscan.TokenArrEnd {
				if !itemCheck(m) {
					return false
				}
			}
			m.checkedValue = m.checkedValue[1:]
			return true
		}
	}
	return nil
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

// Designation creates a textual explanation for the given expression.
func Designation(e gqt.Expression) string {
	switch e := e.(type) {
	case *gqt.ConstrAny:
		return "can be any value"
	case *gqt.ConstrEquals:
		return "must be equal " + Designation(e.Value)
	case *gqt.ConstrNotEquals:
		return "must not be equal " + Designation(e.Value)
	case *gqt.ConstrLenEquals:
		return "length must be equal " + Designation(e.Value)
	case *gqt.ConstrLenNotEquals:
		return "length must not be equal " + Designation(e.Value)
	case *gqt.ConstrLenLess:
		return "length must be less than " + Designation(e.Value)
	case *gqt.ConstrLenGreater:
		return "length must be greater than " + Designation(e.Value)
	case *gqt.ConstrLenLessOrEqual:
		return "length must be less than or equal " + Designation(e.Value)
	case *gqt.ConstrLenGreaterOrEqual:
		return "length must be greater than or equal " + Designation(e.Value)
	case *gqt.ConstrMap:
		return "each item: " + Designation(e.Constraint)
	case *gqt.ConstrLess:
		return "must be less than " + Designation(e.Value)
	case *gqt.ConstrGreater:
		return "must be greater than " + Designation(e.Value)
	case *gqt.ConstrLessOrEqual:
		return "must be less than or equal " + Designation(e.Value)
	case *gqt.ConstrGreaterOrEqual:
		return "must be greater than or equal " + Designation(e.Value)
	case *gqt.ExprParentheses:
		return "(" + Designation(e.Expression) + ")"
	case *gqt.ExprAddition:
		return Designation(e.AddendLeft) + " + " + Designation(e.AddendRight)
	case *gqt.ExprSubtraction:
		return Designation(e.Minuend) + " - " + Designation(e.Subtrahend)
	case *gqt.ExprMultiplication:
		return Designation(e.Multiplicant) + " * " + Designation(e.Multiplicator)
	case *gqt.ExprDivision:
		return Designation(e.Dividend) + " / " + Designation(e.Divisor)
	case *gqt.ExprModulo:
		return Designation(e.Dividend) + " % " + Designation(e.Divisor)
	case *gqt.ExprEqual:
		return Designation(e.Left) + " equal " + Designation(e.Right)
	case *gqt.ExprNotEqual:
		return Designation(e.Left) + " not equal " + Designation(e.Right)
	case *gqt.ExprGreater:
		return Designation(e.Left) + " greater than " + Designation(e.Right)
	case *gqt.ExprLess:
		return Designation(e.Left) + " less than " + Designation(e.Right)
	case *gqt.ExprLessOrEqual:
		return Designation(e.Left) + " less than or equal " + Designation(e.Right)
	case *gqt.ExprGreaterOrEqual:
		return Designation(e.Left) + " greater than or equal " + Designation(e.Right)
	case *gqt.ExprLogicalNegation:
		return "not(" + Designation(e.Expression) + ")"
	case *gqt.ExprNumericNegation:
		return "negative(" + Designation(e.Expression) + ")"
	case *gqt.Number:
		i, ok := e.Int()
		if ok {
			return strconv.FormatInt(int64(i), 10)
		}
		f, _ := e.Float()
		return fmt.Sprintf("%f", f)
	case *gqt.String:
		return fmt.Sprintf("%q", e.Value)
	case *gqt.True:
		return "true"
	case *gqt.False:
		return "false"
	case *gqt.Null:
		return "null"
	case *gqt.Array:
		var b strings.Builder
		b.WriteByte('[')
		for i, v := range e.Items {
			if i > 0 {
				b.WriteString(" ,")
			}
			fmt.Fprintf(&b, "%d: %v", i, Designation(v))
		}
		b.WriteByte(']')
		return b.String()
	case *gqt.Enum:
		return e.Value
	case *gqt.Variable:
		return MakePath(e.Declaration.Parent)
	case *gqt.ExprLogicalOr:
		var b strings.Builder
		for i, v := range e.Expressions {
			if i > 0 {
				b.WriteString(", or ")
			}
			b.WriteString(Designation(v))
		}
		return b.String()
	case *gqt.ExprLogicalAnd:
		var b strings.Builder
		for i, v := range e.Expressions {
			if i > 0 {
				b.WriteString(", and ")
			}
			b.WriteString(Designation(v))
		}
		return b.String()
	case *gqt.Object:
		var b strings.Builder
		b.WriteByte('{')
		for i, v := range e.Fields {
			if i > 0 {
				b.WriteString(" ,")
			}
			fmt.Fprintf(&b, "%q: %v", v.Name.Name, Designation(v.Constraint))
		}
		b.WriteByte('}')
		return b.String()
	}
	panic(fmt.Sprintf("unhandled expression: %T", e))
}

func (m *Checker) expectOrNum(expect *gqlast.Type) (wrongType bool) {
	if expect != nil {
		wrongType, _ = isWrongType(expect, m.checkedValue, m.schema)
		return wrongType
	}
	return m.checkedValue[0].ID != gqlscan.TokenInt &&
		m.checkedValue[0].ID != gqlscan.TokenFloat
}

func (m *Checker) expectOrInt(expect *gqlast.Type) (wrongType bool) {
	if expect != nil {
		wrongType, _ = isWrongType(expect, m.checkedValue, m.schema)
		return wrongType
	}
	return m.checkedValue[0].ID != gqlscan.TokenInt
}

func (m *Checker) expectOrFloat(expect *gqlast.Type) (wrongType bool) {
	if expect != nil {
		wrongType, _ = isWrongType(expect, m.checkedValue, m.schema)
		return wrongType
	}
	return m.checkedValue[0].ID != gqlscan.TokenFloat
}

func (m *Checker) expectOrString(expect *gqlast.Type) (wrongType bool) {
	if expect != nil {
		wrongType, _ = isWrongType(expect, m.checkedValue, m.schema)
		return wrongType
	}
	return m.checkedValue[0].ID != gqlscan.TokenStr &&
		m.checkedValue[0].ID != gqlscan.TokenStrBlock
}

func (m *Checker) expectOrEnum(expect *gqlast.Type) (wrongType bool) {
	if expect != nil {
		wrongType, _ = isWrongType(expect, m.checkedValue, m.schema)
		return wrongType
	}
	return m.checkedValue[0].ID != gqlscan.TokenEnumVal
}

func (m *Checker) expectOrBool(expect *gqlast.Type) (wrongType bool) {
	if expect != nil {
		wrongType, _ = isWrongType(expect, m.checkedValue, m.schema)
		return wrongType
	}
	return m.checkedValue[0].ID != gqlscan.TokenTrue &&
		m.checkedValue[0].ID != gqlscan.TokenFalse
}

func (m *Checker) expectOrHasLen(
	expect *gqlast.Type,
) (wrongType bool) {
	if expect != nil {
		wrongType, _ = isWrongType(expect, m.checkedValue, m.schema)
		return wrongType
	}
	return m.checkedValue[0].ID != gqlscan.TokenArr &&
		m.checkedValue[0].ID != gqlscan.TokenStr &&
		m.checkedValue[0].ID != gqlscan.TokenStrBlock
}

func makeEqArray(
	v *gqt.Array,
	expect *gqlast.Type,
	schema *gqlast.Schema,
) check {
	checks := make([]check, len(v.Items))
	for i := 0; i < len(v.Items); i++ {
		var expect *gqlast.Type
		if expect != nil {
			expect = expect.Elem
		}
		checks[i] = makeCheck(v.Items[i], expect, schema)
	}
	return func(m *Checker) (match bool) {
		if m.checkedValue[0].ID != gqlscan.TokenArr {
			return false
		}
		m.checkedValue = m.checkedValue[1:]
		c := 0
		for ; m.checkedValue[0].ID != gqlscan.TokenArrEnd; c++ {
			if c >= len(checks) {
				return false
			}
			if !checks[c](m) {
				return false
			}
		}
		if c != len(checks) {
			return false
		}
		m.checkedValue = m.checkedValue[1:]
		return true
	}
}

func makeNotEqArray(
	v *gqt.Array,
	expect *gqlast.Type,
	schema *gqlast.Schema,
) check {
	checks := make([]check, len(v.Items))
	for i := 0; i < len(v.Items); i++ {
		var expect *gqlast.Type
		if expect != nil {
			expect = expect.Elem
		}
		checks[i] = makeCheck(v.Items[i], expect, schema)
	}
	return func(m *Checker) (match bool) {
		if m.checkedValue[0].ID != gqlscan.TokenArr {
			return false
		}
		m.checkedValue = m.checkedValue[1:]
		c := 0
		for ; m.checkedValue[0].ID != gqlscan.TokenArrEnd; c++ {
			if c >= len(checks) {
				// Skip all following values to return correct tokenRead
				for m.checkedValue[0].ID != gqlscan.TokenArrEnd {
					l := gqlparse.GetValLen(m.checkedValue)
					m.checkedValue = m.checkedValue[l:]
				}
				return true
			}
			if !checks[c](m) {
				// Skip all following values to return correct tokenRead
				for m.checkedValue[0].ID != gqlscan.TokenArrEnd {
					l := gqlparse.GetValLen(m.checkedValue)
					m.checkedValue = m.checkedValue[l:]
				}
				return true
			}
		}
		m.checkedValue = m.checkedValue[1:]
		// All item checks were matched, finally check length
		return c != len(checks)
	}
}

func makeEqNumber(
	v *gqt.Number,
	expect *gqlast.Type,
	schema *gqlast.Schema,
) check {
	if i, ok := v.Int(); ok {
		return func(m *Checker) (match bool) {
			if m.expectOrInt(expect) {
				return false
			}
			a := atoi.MustI32(m.checkedValue[0].Value)
			m.checkedValue = m.checkedValue[1:]
			return int32(i) == a
		}
	}
	f, _ := v.Float()
	return func(m *Checker) (match bool) {
		if m.expectOrFloat(expect) {
			return false
		}
		a := atoi.MustF64(m.checkedValue[0].Value)
		m.checkedValue = m.checkedValue[1:]
		return f == a
	}
}

func makeNotEqNumber(
	v *gqt.Number,
	expect *gqlast.Type,
	schema *gqlast.Schema,
) check {
	if i, ok := v.Int(); ok {
		return func(m *Checker) (match bool) {
			if m.expectOrInt(expect) {
				return false
			}
			a := atoi.MustI32(m.checkedValue[0].Value)
			m.checkedValue = m.checkedValue[1:]
			return int32(i) != a
		}
	}
	f, _ := v.Float()
	return func(m *Checker) (match bool) {
		if m.expectOrFloat(expect) {
			return false
		}
		a := atoi.MustF64(m.checkedValue[0].Value)
		m.checkedValue = m.checkedValue[1:]
		return f != a
	}
}

func makeEqString(
	v *gqt.String,
	expect *gqlast.Type,
	schema *gqlast.Schema,
) check {
	return func(m *Checker) (match bool) {
		if m.expectOrString(expect) {
			return false
		}
		t := m.checkedValue[0]
		m.checkedValue = m.checkedValue[1:]
		switch t.ID {
		case gqlscan.TokenStr:
			return v.Value == string(t.Value)
		case gqlscan.TokenStrBlock:
			return v.Value == string(t.Value)
		}
		return false
	}
}

func makeNotEqString(
	v *gqt.String,
	expect *gqlast.Type,
	schema *gqlast.Schema,
) check {
	return func(m *Checker) (match bool) {
		if m.expectOrString(expect) {
			return false
		}
		t := m.checkedValue[0]
		m.checkedValue = m.checkedValue[1:]
		switch t.ID {
		case gqlscan.TokenStr:
			return v.Value != string(t.Value)
		case gqlscan.TokenStrBlock:
			return v.Value != string(t.Value)
		}
		return false
	}
}

func makeEqEnum(
	v *gqt.Enum,
	expect *gqlast.Type,
	schema *gqlast.Schema,
) check {
	return func(m *Checker) (match bool) {
		if m.expectOrEnum(expect) {
			return false
		}
		b := m.checkedValue[0].Value
		m.checkedValue = m.checkedValue[1:]
		return v.Value == string(b)
	}
}

func makeNotEqEnum(
	v *gqt.Enum,
	expect *gqlast.Type,
	schema *gqlast.Schema,
) check {
	return func(m *Checker) (match bool) {
		if m.expectOrEnum(expect) {
			return false
		}
		b := m.checkedValue[0].Value
		m.checkedValue = m.checkedValue[1:]
		return v.Value != string(b)
	}
}

func makeEqNull(
	expect *gqlast.Type,
	schema *gqlast.Schema,
) check {
	return func(m *Checker) (match bool) {
		if wrong, _ := isWrongType(expect, m.checkedValue, schema); wrong {
			return false
		}
		t := m.checkedValue[0].ID
		m.checkedValue = m.checkedValue[1:]
		return t == gqlscan.TokenNull
	}
}

func makeNotEqNull(
	expect *gqlast.Type,
	schema *gqlast.Schema,
) check {
	return func(m *Checker) (match bool) {
		if wrong, _ := isWrongType(expect, m.checkedValue, schema); wrong {
			return false
		}
		t := m.checkedValue[0].ID
		m.checkedValue = m.checkedValue[1:]
		return t != gqlscan.TokenNull
	}
}

func makeEqBool(
	value bool,
	expect *gqlast.Type,
	schema *gqlast.Schema,
) check {
	v := gqlscan.TokenFalse
	if value {
		v = gqlscan.TokenTrue
	}
	return func(m *Checker) (match bool) {
		if m.expectOrBool(expect) {
			return false
		}
		t := m.checkedValue[0].ID
		m.checkedValue = m.checkedValue[1:]
		return t == v
	}
}

func makeNotEqBool(
	value bool,
	expect *gqlast.Type,
	schema *gqlast.Schema,
) check {
	v := gqlscan.TokenFalse
	if value {
		v = gqlscan.TokenTrue
	}
	return func(m *Checker) (match bool) {
		if m.expectOrBool(expect) {
			return false
		}
		t := m.checkedValue[0].ID
		m.checkedValue = m.checkedValue[1:]
		return t != v
	}
}

func makeEqObject(
	v *gqt.Object,
	expect *gqlast.Type,
	schema *gqlast.Schema,
) check {
	checks := make(map[string]check, len(v.Fields))
	fieldChecked := make(map[string]bool, len(v.Fields))
	var requiredChecks int
	for _, v := range v.Fields {
		var expect *gqlast.Type
		if v.Def != nil {
			expect = v.Def.Type
			if expect.NonNull {
				requiredChecks++
			}
		}
		checks[v.Name.Name] = makeCheck(v.Constraint, expect, schema)
		fieldChecked[v.Name.Name] = false
	}
	return func(m *Checker) (match bool) {
		if wrong, _ := isWrongType(expect, m.checkedValue, schema); wrong {
			return false
		}
		if m.checkedValue[0].ID != gqlscan.TokenObj {
			return false
		}
		m.checkedValue = m.checkedValue[1:]

		// Reset check status
		for k := range fieldChecked {
			fieldChecked[k] = false
		}

		c := 0
		for ; m.checkedValue[0].ID != gqlscan.TokenObjEnd; c++ {
			if c >= len(checks) {
				return false
			}

			check := checks[string(m.checkedValue[0].Value)]
			if check == nil {
				// Unknown field, wrong object type
				return false
			}

			if fieldChecked[string(m.checkedValue[0].Value)] {
				// Field provided twice, invalid object
				return false
			}
			fieldChecked[string(m.checkedValue[0].Value)] = true

			m.checkedValue = m.checkedValue[1:]

			if !check(m) {
				return false
			}
		}
		if requiredChecks < 1 && c != len(checks) || c < requiredChecks {
			// Not all required fields were provided, wrong object type
			return false
		}
		m.checkedValue = m.checkedValue[1:]
		return true
	}
}

func makeNotEqObject(
	v *gqt.Object,
	expect *gqlast.Type,
	schema *gqlast.Schema,
) check {
	checks := make(map[string]check, len(v.Fields))
	fieldChecked := make(map[string]bool, len(v.Fields))
	var requiredChecks int
	for _, v := range v.Fields {
		var expect *gqlast.Type
		if v.Def != nil {
			expect = v.Def.Type
			if expect.NonNull {
				requiredChecks++
			}
		}
		checks[v.Name.Name] = makeCheck(v.Constraint, expect, schema)
		fieldChecked[v.Name.Name] = false
	}
	return func(m *Checker) (match bool) {
		if wrong, _ := isWrongType(expect, m.checkedValue, schema); wrong {
			return false
		}
		if m.checkedValue[0].ID != gqlscan.TokenObj {
			return false
		}
		m.checkedValue = m.checkedValue[1:]

		// Reset check status
		for k := range fieldChecked {
			fieldChecked[k] = false
		}

		c := 0
		for ; m.checkedValue[0].ID != gqlscan.TokenObjEnd; c++ {
			if c >= len(checks) {
				return false
			}

			check := checks[string(m.checkedValue[0].Value)]
			if check == nil {
				// Unknown field, wrong object type
				return false
			}

			if fieldChecked[string(m.checkedValue[0].Value)] {
				// Field provided twice, invalid object
				return false
			}
			fieldChecked[string(m.checkedValue[0].Value)] = true

			m.checkedValue = m.checkedValue[1:]

			if check(m) {
				// Don't return just yet as we're not sure whether the
				// object type was correct.
				match = true
			}
		}
		if requiredChecks < 1 && c != len(checks) || c < requiredChecks {
			// Not all required fields were provided, wrong object type
			return false
		}
		m.checkedValue = m.checkedValue[1:]
		return !match
	}
}

func makeEqExpression(
	v gqt.Expression,
	expect *gqlast.Type,
	schema *gqlast.Schema,
) check {
	return func(m *Checker) (match bool) {
		if wrong, _ := isWrongType(expect, m.checkedValue, schema); wrong {
			return false
		}
		m.resolveExpr(v)
		u := m.popStack()

		switch u.Type() {
		case union.TypeNull:
			return m.checkedValue[0].ID == gqlscan.TokenNull
		case union.TypeBoolean:
			b, _ := u.Bool()
			if b {
				return m.checkedValue[0].ID == gqlscan.TokenTrue
			}
			return m.checkedValue[0].ID == gqlscan.TokenFalse
		case union.TypeInt:
			if m.checkedValue[0].ID == gqlscan.TokenInt {
				u, _ := u.Int()
				return u == atoi.MustI32(m.checkedValue[0].Value)
			}
			return false
		case union.TypeFloat:
			if m.checkedValue[0].ID == gqlscan.TokenFloat {
				u, _ := u.Float()
				return u == atoi.MustF64(m.checkedValue[0].Value)
			}
			return false
		case union.TypeString:
			if m.checkedValue[0].ID == gqlscan.TokenStr {
				u, _ := u.String()
				return u == string(m.checkedValue[0].Value)
			}
			return false
		case union.TypeEnum:
			if m.checkedValue[0].ID == gqlscan.TokenEnumVal {
				u, _ := u.Enum()
				return u == string(m.checkedValue[0].Value)
			}
			return false
		case union.TypeTokens:
			if len(u.Tokens()) != len(m.checkedValue) {
				return false
			}
			for i := range m.checkedValue {
				if m.checkedValue[i].ID != u.Tokens()[i].ID {
					return false
				}
				switch m.checkedValue[i].ID {
				case gqlscan.TokenStrBlock:
					if string(m.checkedValue[i].Value) != string(u.Tokens()[i].Value) {
						return false
					}
				default:
					if string(m.checkedValue[i].Value) != string(u.Tokens()[i].Value) {
						return false
					}
				}
			}
		}

		return true
	}
}

func unionsEqual(a, b []union.Union) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !union.Equal(a[i], b[i]) {
			return false
		}
	}
	return true
}
