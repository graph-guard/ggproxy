package constrcheck

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/graph-guard/ggproxy/engine/playmon/internal/constrcheck/internal/union"
	"github.com/graph-guard/ggproxy/engine/playmon/internal/pathscan"
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
	pathByVar map[*gqt.Variable]string

	// variableValues keys will be set at initialization
	// but values must be set before every call to Check.
	variableValues map[string][]gqlparse.Token

	// inputValue is set before every call to Check.
	inputValue []gqlparse.Token
	// checkedValue is set before every call to Check.
	checkedValue []gqlparse.Token

	// stack is reset in every call to Check.
	stack []union.Union
}

func (c *Checker) SetVariableValue(path []byte, value []gqlparse.Token) {
	c.variableValues[string(path)] = value
}

func (c *Checker) Init(inputValue []gqlparse.Token) {
	c.inputValue = inputValue
}

func (c *Checker) stackTopType() union.Type {
	return c.stack[len(c.stack)-1].Type()
}

func (c *Checker) popStack() union.Union {
	t := c.stack[len(c.stack)-1]
	c.stack = c.stack[:len(c.stack)-1]
	return t
}

func (c *Checker) popStackArrays() (left, right []union.Union) {
	if c.stack[len(c.stack)-1].Type() != union.TypeArray {
		return
	}
	i := len(c.stack) - 2
	var count int
LOOP_R:
	for l := 1; i > 0; i, count = i-1, count+1 {
		switch c.stack[i].Type() {
		case union.TypeArray:
			l++
		case union.TypeArrayEnd:
			l--
			if l < 1 {
				break LOOP_R
			}
		}
	}
	right = c.stack[i+1 : len(c.stack)-1]

	i--
	iv := i
	if c.stack[i].Type() != union.TypeArray {
		return
	}
	count = 0
	i--
LOOP_L:
	for l := 1; i > 0; i, count = i-1, count+1 {
		switch c.stack[i].Type() {
		case union.TypeArray:
			l++
		case union.TypeArrayEnd:
			l--
			if l < 1 {
				break LOOP_L
			}
		}
	}

	left = c.stack[i+1 : iv]
	c.stack = c.stack[:i]
	return left, right
}

func (c *Checker) pushStackArray() {
	c.stack = append(c.stack, union.Array())
}

func (c *Checker) pushStackArrayEnd() {
	c.stack = append(c.stack, union.ArrayEnd())
}

func (c *Checker) pushStackInt(v int32) {
	c.stack = append(c.stack, union.Int(v))
}

func (c *Checker) pushStackFloat(v float64) {
	c.stack = append(c.stack, union.Float(v))
}

func (c *Checker) pushStackBool(v bool) {
	if v {
		c.stack = append(c.stack, union.True())
		return
	}
	c.stack = append(c.stack, union.False())
}

// Check returns true if the value for the given path is accepted,
// otherwise returns false.
func (c *Checker) Check(path string) bool {
	c.stack = c.stack[:0]
	cf := c.checkers[path]
	if cf == nil {
		return false
	}
	v := c.variableValues[path]
	c.inputValue, c.checkedValue = v, v
	return cf(c)
}

// check is a constraint check function which returns true
// if the input value matches the constraint and can be accepted,
// otherwise returns false.
type check func(*Checker) (match bool)

// resolveExpr resolves expression e into a union and pushes it onto the stack.
func (c *Checker) resolveExpr(e gqt.Expression) union.Type {
	switch e := e.(type) {
	case *gqt.Array:
		c.pushStackArrayEnd()
		for i := len(e.Items) - 1; i >= 0; i-- {
			c.resolveExpr(e.Items[i].(*gqt.ConstrEquals).Value)
		}
		c.pushStackArray()
		return union.TypeArray
	case *gqt.Variable:
		p, ok := c.pathByVar[e]
		if !ok {
			if s := c.variableValues[p]; s != nil {
				c.stack = append(c.stack, union.Tokens(s))
				return union.TypeTokens
			}
		}
		c.stack = append(c.stack, union.Null())
		return union.TypeNull
	case *gqt.Number:
		if i, ok := e.Int(); ok {
			c.stack = append(c.stack, union.Int(int32(i)))
			return union.TypeInt
		}
		f, _ := e.Float()
		c.stack = append(c.stack, union.Float(f))
		return union.TypeFloat
	case *gqt.True:
		c.stack = append(c.stack, union.True())
		return union.TypeBoolean
	case *gqt.False:
		c.stack = append(c.stack, union.False())
		return union.TypeBoolean
	case *gqt.Enum:
		c.stack = append(c.stack, union.Enum(e.Value))
		return union.TypeEnum
	case *gqt.String:
		c.stack = append(c.stack, union.String(e.Value))
		return union.TypeString
	case *gqt.Null:
		return union.TypeNull
	case *gqt.ExprAddition:
		c.resolveExpr(e.AddendLeft)
		c.resolveExpr(e.AddendRight)
		r := c.popStack()
		l := c.popStack()
		if l.Type() == union.TypeInt && r.Type() == union.TypeInt {
			l, _ := l.Int()
			r, _ := r.Int()
			c.pushStackInt(l + r)
			return union.TypeInt
		}
		{
			l, _ := l.Float()
			r, _ := r.Float()
			c.pushStackFloat(l + r)
		}
		return union.TypeFloat
	case *gqt.ExprSubtraction:
		c.resolveExpr(e.Minuend)
		c.resolveExpr(e.Subtrahend)
		r := c.popStack()
		l := c.popStack()
		if l.Type() == union.TypeInt && r.Type() == union.TypeInt {
			l, _ := l.Int()
			r, _ := r.Int()
			c.pushStackInt(l - r)
			return union.TypeInt
		}
		{
			l, _ := l.Float()
			r, _ := r.Float()
			c.pushStackFloat(l - r)
		}
		return union.TypeFloat
	case *gqt.ExprMultiplication:
		c.resolveExpr(e.Multiplicant)
		c.resolveExpr(e.Multiplicator)
		r := c.popStack()
		l := c.popStack()
		if l.Type() == union.TypeInt && r.Type() == union.TypeInt {
			l, _ := l.Int()
			r, _ := r.Int()
			c.pushStackInt(l * r)
			return union.TypeInt
		}
		{
			l, _ := l.Float()
			r, _ := r.Float()
			c.pushStackFloat(l * r)
		}
		return union.TypeFloat
	case *gqt.ExprDivision:
		c.resolveExpr(e.Dividend)
		c.resolveExpr(e.Divisor)
		r := c.popStack()
		l := c.popStack()
		if l.Type() == union.TypeInt && r.Type() == union.TypeInt {
			l, _ := l.Int()
			r, _ := r.Int()
			c.pushStackInt(l / r)
			return union.TypeInt
		}
		{
			l, _ := l.Float()
			r, _ := r.Float()
			c.pushStackFloat(l / r)
		}
		return union.TypeFloat
	case *gqt.ExprModulo:
		c.resolveExpr(e.Dividend)
		c.resolveExpr(e.Divisor)
		r := c.popStack()
		l := c.popStack()
		if l.Type() == union.TypeInt && r.Type() == union.TypeInt {
			l, _ := l.Int()
			r, _ := r.Int()
			c.pushStackInt(l % r)
			return union.TypeInt
		}
		{
			l, _ := l.Float()
			r, _ := r.Float()
			c.pushStackFloat(math.Mod(l, r))
		}
		return union.TypeFloat
	case *gqt.ExprNumericNegation:
		c.resolveExpr(e.Expression)
		i := c.popStack()
		switch i.Type() {
		case union.TypeFloat:
			i, _ := i.Float()
			c.pushStackFloat(-i)
			return union.TypeFloat
		case union.TypeInt:
			i, _ := i.Int()
			c.pushStackInt(-i)
			return union.TypeInt
		case union.TypeTokens:
			switch i.Tokens()[0].ID {
			case gqlscan.TokenInt:
				i := atoi.MustI32(i.Tokens()[0].Value)
				c.pushStackInt(-i)
				return union.TypeInt
			case gqlscan.TokenFloat:
				f := atoi.MustF64(i.Tokens()[0].Value)
				c.pushStackFloat(-f)
				return union.TypeFloat
			}
			panic(fmt.Errorf("unexpected token type: %q", i.Tokens()[0].ID.String()))
		}
		panic(fmt.Errorf("unexpected union type: %q", i.Type().String()))
	case *gqt.ExprEqual:
		c.resolveExpr(e.Left)
		c.resolveExpr(e.Right)
		equal := false
		if c.stackTopType() == union.TypeArray {
			left, right := c.popStackArrays()
			equal = unionsEqual(left, right)
		} else {
			r := c.popStack()
			l := c.popStack()
			equal = union.Equal(l, r)
		}
		c.pushStackBool(equal)
		return union.TypeBoolean
	case *gqt.ExprNotEqual:
		c.resolveExpr(e.Left)
		c.resolveExpr(e.Right)
		notEqual := false
		if c.stackTopType() == union.TypeArray {
			left, right := c.popStackArrays()
			notEqual = !unionsEqual(left, right)
		} else {
			r := c.popStack()
			l := c.popStack()
			notEqual = !union.Equal(l, r)
		}
		c.pushStackBool(notEqual)
		return union.TypeBoolean
	case *gqt.ExprLogicalNegation:
		c.resolveExpr(e.Expression)
		u := c.popStack()
		b, _ := u.Bool()
		c.pushStackBool(!b)
		return union.TypeBoolean
	case *gqt.ExprGreater:
		c.resolveExpr(e.Left)
		c.resolveExpr(e.Right)
		r := c.popStack()
		l := c.popStack()
		if l.Type() == union.TypeInt && r.Type() == union.TypeInt {
			l, _ := l.Int()
			r, _ := r.Int()
			c.pushStackBool(l > r)
		} else {
			l, _ := l.Float()
			r, _ := r.Float()
			c.pushStackBool(l > r)
		}
		return union.TypeBoolean
	case *gqt.ExprLess:
		c.resolveExpr(e.Left)
		c.resolveExpr(e.Right)
		r := c.popStack()
		l := c.popStack()
		if l.Type() == union.TypeInt && r.Type() == union.TypeInt {
			l, _ := l.Int()
			r, _ := r.Int()
			c.pushStackBool(l < r)
		} else {
			l, _ := l.Float()
			r, _ := r.Float()
			c.pushStackBool(l < r)
		}
		return union.TypeBoolean
	case *gqt.ExprGreaterOrEqual:
		c.resolveExpr(e.Left)
		c.resolveExpr(e.Right)
		r := c.popStack()
		l := c.popStack()
		if l.Type() == union.TypeInt && r.Type() == union.TypeInt {
			l, _ := l.Int()
			r, _ := r.Int()
			c.pushStackBool(l >= r)
		} else {
			l, _ := l.Float()
			r, _ := r.Float()
			c.pushStackBool(l >= r)
		}
		return union.TypeBoolean
	case *gqt.ExprLessOrEqual:
		c.resolveExpr(e.Left)
		c.resolveExpr(e.Right)
		r := c.popStack()
		l := c.popStack()
		if l.Type() == union.TypeInt && r.Type() == union.TypeInt {
			l, _ := l.Int()
			r, _ := r.Int()
			c.pushStackBool(l <= r)
		} else {
			l, _ := l.Float()
			r, _ := r.Float()
			c.pushStackBool(l <= r)
		}
		return union.TypeBoolean
	case *gqt.ExprLogicalOr:
		for _, x := range e.Expressions {
			c.resolveExpr(x)
			u := c.popStack()
			b, _ := u.Bool()
			if b {
				c.pushStackBool(true)
				return union.TypeBoolean
			}
		}
		c.pushStackBool(false)
		return union.TypeBoolean
	case *gqt.ExprLogicalAnd:
		for _, x := range e.Expressions {
			c.resolveExpr(x)
			u := c.popStack()
			b, _ := u.Bool()
			if !b {
				c.pushStackBool(false)
				return union.TypeBoolean
			}
		}
		c.pushStackBool(true)
		return union.TypeBoolean
	case *gqt.ExprParentheses:
		return c.resolveExpr(e.Expression)
	}
	panic(fmt.Errorf("unhandled value expression type: %T", e))
}

// New creates a constraint checker instance for each path of o.
func New(o *gqt.Operation, s *gqlast.Schema) *Checker {
	c := &Checker{
		operation:      o,
		schema:         s,
		checkers:       make(map[string]check),
		stack:          make([]union.Union, 1024),
		variableValues: make(map[string][]gqlparse.Token),
		pathByVar:      make(map[*gqt.Variable]string),
	}
	pathscan.InAST(
		o,
		func(path []byte, e gqt.Expression) (stop bool) {
			// On structural
			return false
		}, func(path []byte, e gqt.Expression) (stop bool) {
			// On argument
			a := e.(*gqt.Argument)
			var expect *gqlast.Type
			if a.Def != nil {
				expect = a.Def.Type
			}
			if fn := makeCheck(a.Constraint, expect, s); fn != nil {
				c.checkers[string(path)] = fn
			}
			return false
		},
		func(path []byte, e gqt.Expression) (stop bool) {
			// On variable
			a := e.(*gqt.Variable)
			c.pathByVar[a] = string(path)
			c.variableValues[string(path)] = nil // Set key
			return false
		},
	)
	return c
}

func makeCheck(
	e gqt.Expression,
	expect *gqlast.Type,
	schema *gqlast.Schema,
) check {
	switch e := e.(type) {
	case *gqt.ConstrAny:
		return func(c *Checker) (match bool) {
			if wrong, _ := isWrongType(expect, c.checkedValue, schema); wrong {
				return false
			}

			// Make sure the value is semantically valid.
			return !gqlparse.ScanValuesInArrays(c.checkedValue, func(t []gqlparse.Token) (stop bool) {
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
		return func(c *Checker) (match bool) {
			r := c.checkedValue
			for _, e := range exprCheckers {
				if e(c) {
					return true
				}
				// Reset value to start checking from the same index
				c.checkedValue = r
			}
			return false
		}
	case *gqt.ExprLogicalAnd:
		exprCheckers := make([]check, len(e.Expressions))
		for i, e := range e.Expressions {
			exprCheckers[i] = makeCheck(e, expect, schema)
		}
		return func(c *Checker) (match bool) {
			r := c.checkedValue
			for _, e := range exprCheckers {
				if !e(c) {
					return false
				}
				// Reset value to start checking from the same index
				c.checkedValue = r
			}
			return true
		}
	case *gqt.ConstrGreater:
		return func(c *Checker) (match bool) {
			if c.expectOrNum(expect) {
				return false
			}
			c.resolveExpr(e.Value)
			u := c.popStack()
			switch c.checkedValue[0].ID {
			case gqlscan.TokenInt:
				i := atoi.MustI32(c.checkedValue[0].Value)
				if u, ok := u.Int(); ok != union.ValueNone {
					match = i > u
					break
				}
				u, _ := u.Float()
				match = float64(i) > u
			case gqlscan.TokenFloat:
				i := atoi.MustF64(c.checkedValue[0].Value)
				u, _ := u.Float()
				match = i > u
			}
			c.checkedValue = c.checkedValue[1:]
			return match
		}
	case *gqt.ConstrGreaterOrEqual:
		return func(c *Checker) (match bool) {
			if c.expectOrNum(expect) {
				return false
			}
			c.resolveExpr(e.Value)
			u := c.popStack()
			switch c.checkedValue[0].ID {
			case gqlscan.TokenInt:
				i := atoi.MustI32(c.checkedValue[0].Value)
				if u, ok := u.Int(); ok != union.ValueNone {
					match = i >= u
					break
				}
				u, _ := u.Float()
				match = float64(i) >= u
			case gqlscan.TokenFloat:
				i := atoi.MustF64(c.checkedValue[0].Value)
				u, _ := u.Float()
				match = i >= u
			}
			c.checkedValue = c.checkedValue[1:]
			return match
		}
	case *gqt.ConstrLess:
		return func(c *Checker) (match bool) {
			if c.expectOrNum(expect) {
				return false
			}
			c.resolveExpr(e.Value)
			u := c.popStack()
			switch c.checkedValue[0].ID {
			case gqlscan.TokenInt:
				i := atoi.MustI32(c.checkedValue[0].Value)
				if u, ok := u.Int(); ok != union.ValueNone {
					match = i < u
					break
				}
				u, _ := u.Float()
				match = float64(i) < u
			case gqlscan.TokenFloat:
				i := atoi.MustF64(c.checkedValue[0].Value)
				u, _ := u.Float()
				match = i < u
			}
			c.checkedValue = c.checkedValue[1:]
			return match
		}
	case *gqt.ConstrLessOrEqual:
		return func(c *Checker) (match bool) {
			if c.expectOrNum(expect) {
				return false
			}
			c.resolveExpr(e.Value)
			u := c.popStack()
			switch c.checkedValue[0].ID {
			case gqlscan.TokenInt:
				i := atoi.MustI32(c.checkedValue[0].Value)
				if u, ok := u.Int(); ok != union.ValueNone {
					match = i <= u
					break
				}
				u, _ := u.Float()
				match = float64(i) <= u
			case gqlscan.TokenFloat:
				i := atoi.MustF64(c.checkedValue[0].Value)
				u, _ := u.Float()
				match = i <= u
			}
			c.checkedValue = c.checkedValue[1:]
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
		return func(c *Checker) (match bool) {
			return !fn(c)
		}
	case *gqt.ConstrLenEquals:
		return func(c *Checker) (match bool) {
			if c.expectOrHasLen(expect) {
				return false
			}

			var length int
			switch c.checkedValue[0].ID {
			case gqlscan.TokenArr:
				var tokensRead int
				length, tokensRead = gqlparse.CountValuesUntil(
					c.checkedValue[1:],
					gqlscan.TokenArrEnd,
				)
				c.checkedValue = c.checkedValue[tokensRead:]
			case gqlscan.TokenStr:
				length = len(c.checkedValue[0].Value)
			case gqlscan.TokenStrBlock:
				length = len(c.checkedValue[0].Value)
			}

			c.resolveExpr(e.Value)
			u := c.popStack()
			if u, ok := u.Int(); ok != union.ValueNone {
				return int32(length) == u
			}
			if u, ok := u.Float(); ok != union.ValueNone {
				return float64(length) == u
			}
			panic(fmt.Errorf("unexpected value type: %s", u.Type().String()))
		}
	case *gqt.ConstrLenNotEquals:
		return func(c *Checker) (match bool) {
			if c.expectOrHasLen(expect) {
				return false
			}

			var length int
			switch c.checkedValue[0].ID {
			case gqlscan.TokenArr:
				var tokensRead int
				length, tokensRead = gqlparse.CountValuesUntil(
					c.checkedValue[1:],
					gqlscan.TokenArrEnd,
				)
				c.checkedValue = c.checkedValue[tokensRead:]
			case gqlscan.TokenStr:
				length = len(c.checkedValue[0].Value)
			case gqlscan.TokenStrBlock:
				length = len(c.checkedValue[0].Value)
			}

			c.resolveExpr(e.Value)
			u := c.popStack()
			if u, ok := u.Int(); ok != union.ValueNone {
				return int32(length) != u
			}
			if u, ok := u.Float(); ok != union.ValueNone {
				return float64(length) != u
			}
			panic(fmt.Errorf("unexpected value type: %s", u.Type().String()))
		}
	case *gqt.ConstrLenGreater:
		return func(c *Checker) (match bool) {
			if c.expectOrHasLen(expect) {
				return false
			}

			var length int
			switch c.checkedValue[0].ID {
			case gqlscan.TokenArr:
				var tokensRead int
				length, tokensRead = gqlparse.CountValuesUntil(
					c.checkedValue[1:],
					gqlscan.TokenArrEnd,
				)
				c.checkedValue = c.checkedValue[tokensRead:]
			case gqlscan.TokenStr:
				length = len(c.checkedValue[0].Value)
			case gqlscan.TokenStrBlock:
				length = len(c.checkedValue[0].Value)
			}

			c.resolveExpr(e.Value)
			u := c.popStack()
			if u, ok := u.Int(); ok != union.ValueNone {
				return int32(length) > u
			}
			if u, ok := u.Float(); ok != union.ValueNone {
				return float64(length) > u
			}
			panic(fmt.Errorf("unexpected value type: %s", u.Type().String()))
		}
	case *gqt.ConstrLenGreaterOrEqual:
		return func(c *Checker) (match bool) {
			if c.expectOrHasLen(expect) {
				return false
			}

			var length int
			switch c.checkedValue[0].ID {
			case gqlscan.TokenArr:
				var tokensRead int
				length, tokensRead = gqlparse.CountValuesUntil(
					c.checkedValue[1:],
					gqlscan.TokenArrEnd,
				)
				c.checkedValue = c.checkedValue[tokensRead:]
			case gqlscan.TokenStr:
				length = len(c.checkedValue[0].Value)
			case gqlscan.TokenStrBlock:
				length = len(c.checkedValue[0].Value)
			}

			c.resolveExpr(e.Value)
			u := c.popStack()
			if u, ok := u.Int(); ok != union.ValueNone {
				return int32(length) >= u
			}
			if u, ok := u.Float(); ok != union.ValueNone {
				return float64(length) >= u
			}
			panic(fmt.Errorf("unexpected value type: %s", u.Type().String()))
		}
	case *gqt.ConstrLenLess:
		return func(c *Checker) (match bool) {
			if c.expectOrHasLen(expect) {
				return false
			}

			var length int
			switch c.checkedValue[0].ID {
			case gqlscan.TokenArr:
				var tokensRead int
				length, tokensRead = gqlparse.CountValuesUntil(
					c.checkedValue[1:],
					gqlscan.TokenArrEnd,
				)
				c.checkedValue = c.checkedValue[tokensRead:]
			case gqlscan.TokenStr:
				length = len(c.checkedValue[0].Value)
			case gqlscan.TokenStrBlock:
				length = len(c.checkedValue[0].Value)
			}

			c.resolveExpr(e.Value)
			u := c.popStack()
			if u, ok := u.Int(); ok != union.ValueNone {
				return int32(length) < u
			}
			if u, ok := u.Float(); ok != union.ValueNone {
				return float64(length) < u
			}
			panic(fmt.Errorf("unexpected value type: %s", u.Type().String()))
		}
	case *gqt.ConstrLenLessOrEqual:
		return func(c *Checker) (match bool) {
			if c.expectOrHasLen(expect) {
				return false
			}

			var length int
			switch c.checkedValue[0].ID {
			case gqlscan.TokenArr:
				var tokensRead int
				length, tokensRead = gqlparse.CountValuesUntil(
					c.checkedValue[1:],
					gqlscan.TokenArrEnd,
				)
				c.checkedValue = c.checkedValue[tokensRead:]
			case gqlscan.TokenStr:
				length = len(c.checkedValue[0].Value)
			case gqlscan.TokenStrBlock:
				length = len(c.checkedValue[0].Value)
			}

			c.resolveExpr(e.Value)
			u := c.popStack()
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
		return func(c *Checker) (match bool) {
			if wrong, _ := isWrongType(expect, c.checkedValue, schema); wrong {
				return false
			}
			for c.checkedValue[0].ID != gqlscan.TokenArr {
				return false
			}
			c.checkedValue = c.checkedValue[1:]
			for c.checkedValue[0].ID != gqlscan.TokenArrEnd {
				if !itemCheck(c) {
					return false
				}
			}
			c.checkedValue = c.checkedValue[1:]
			return true
		}
	}
	return nil
}

// Designation creates a textual explanation for the given expression.
func Designation(c *Checker, e gqt.Expression) string {
	switch e := e.(type) {
	case *gqt.ConstrAny:
		return "can be any value"
	case *gqt.ConstrEquals:
		return "must be equal " + Designation(c, e.Value)
	case *gqt.ConstrNotEquals:
		return "must not be equal " + Designation(c, e.Value)
	case *gqt.ConstrLenEquals:
		return "length must be equal " + Designation(c, e.Value)
	case *gqt.ConstrLenNotEquals:
		return "length must not be equal " + Designation(c, e.Value)
	case *gqt.ConstrLenLess:
		return "length must be less than " + Designation(c, e.Value)
	case *gqt.ConstrLenGreater:
		return "length must be greater than " + Designation(c, e.Value)
	case *gqt.ConstrLenLessOrEqual:
		return "length must be less than or equal " + Designation(c, e.Value)
	case *gqt.ConstrLenGreaterOrEqual:
		return "length must be greater than or equal " + Designation(c, e.Value)
	case *gqt.ConstrMap:
		return "each item: " + Designation(c, e.Constraint)
	case *gqt.ConstrLess:
		return "must be less than " + Designation(c, e.Value)
	case *gqt.ConstrGreater:
		return "must be greater than " + Designation(c, e.Value)
	case *gqt.ConstrLessOrEqual:
		return "must be less than or equal " + Designation(c, e.Value)
	case *gqt.ConstrGreaterOrEqual:
		return "must be greater than or equal " + Designation(c, e.Value)
	case *gqt.ExprParentheses:
		return "(" + Designation(c, e.Expression) + ")"
	case *gqt.ExprAddition:
		return Designation(c, e.AddendLeft) +
			" + " +
			Designation(c, e.AddendRight)
	case *gqt.ExprSubtraction:
		return Designation(c, e.Minuend) +
			" - " +
			Designation(c, e.Subtrahend)
	case *gqt.ExprMultiplication:
		return Designation(c, e.Multiplicant) +
			" * " +
			Designation(c, e.Multiplicator)
	case *gqt.ExprDivision:
		return Designation(c, e.Dividend) +
			" / " +
			Designation(c, e.Divisor)
	case *gqt.ExprModulo:
		return Designation(c, e.Dividend) +
			" % " +
			Designation(c, e.Divisor)
	case *gqt.ExprEqual:
		return Designation(c, e.Left) +
			" equal " +
			Designation(c, e.Right)
	case *gqt.ExprNotEqual:
		return Designation(c, e.Left) +
			" not equal " +
			Designation(c, e.Right)
	case *gqt.ExprGreater:
		return Designation(c, e.Left) +
			" greater than " +
			Designation(c, e.Right)
	case *gqt.ExprLess:
		return Designation(c, e.Left) +
			" less than " +
			Designation(c, e.Right)
	case *gqt.ExprLessOrEqual:
		return Designation(c, e.Left) +
			" less than or equal " +
			Designation(c, e.Right)
	case *gqt.ExprGreaterOrEqual:
		return Designation(c, e.Left) +
			" greater than or equal " +
			Designation(c, e.Right)
	case *gqt.ExprLogicalNegation:
		return "not(" + Designation(c, e.Expression) + ")"
	case *gqt.ExprNumericNegation:
		return "negative(" + Designation(c, e.Expression) + ")"
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
			fmt.Fprintf(&b, "%d: %v", i, Designation(c, v))
		}
		b.WriteByte(']')
		return b.String()
	case *gqt.Enum:
		return e.Value
	case *gqt.Variable:
		p, ok := c.pathByVar[e]
		if !ok {
			return p
		}
	case *gqt.ExprLogicalOr:
		var b strings.Builder
		for i, v := range e.Expressions {
			if i > 0 {
				b.WriteString(", or ")
			}
			b.WriteString(Designation(c, v))
		}
		return b.String()
	case *gqt.ExprLogicalAnd:
		var b strings.Builder
		for i, v := range e.Expressions {
			if i > 0 {
				b.WriteString(", and ")
			}
			b.WriteString(Designation(c, v))
		}
		return b.String()
	case *gqt.Object:
		var b strings.Builder
		b.WriteByte('{')
		for i, v := range e.Fields {
			if i > 0 {
				b.WriteString(" ,")
			}
			fmt.Fprintf(&b, "%q: %v", v.Name.Name, Designation(c, v.Constraint))
		}
		b.WriteByte('}')
		return b.String()
	}
	panic(fmt.Sprintf("unhandled expression: %T", e))
}

func (c *Checker) expectOrNum(expect *gqlast.Type) (wrongType bool) {
	if expect != nil {
		wrongType, _ = isWrongType(expect, c.checkedValue, c.schema)
		return wrongType
	}
	return c.checkedValue[0].ID != gqlscan.TokenInt &&
		c.checkedValue[0].ID != gqlscan.TokenFloat
}

func (c *Checker) expectOrInt(expect *gqlast.Type) (wrongType bool) {
	if expect != nil {
		wrongType, _ = isWrongType(expect, c.checkedValue, c.schema)
		return wrongType
	}
	return c.checkedValue[0].ID != gqlscan.TokenInt
}

func (c *Checker) expectOrFloat(expect *gqlast.Type) (wrongType bool) {
	if expect != nil {
		wrongType, _ = isWrongType(expect, c.checkedValue, c.schema)
		return wrongType
	}
	return c.checkedValue[0].ID != gqlscan.TokenFloat
}

func (c *Checker) expectOrString(expect *gqlast.Type) (wrongType bool) {
	if expect != nil {
		wrongType, _ = isWrongType(expect, c.checkedValue, c.schema)
		return wrongType
	}
	return c.checkedValue[0].ID != gqlscan.TokenStr &&
		c.checkedValue[0].ID != gqlscan.TokenStrBlock
}

func (c *Checker) expectOrEnum(expect *gqlast.Type) (wrongType bool) {
	if expect != nil {
		wrongType, _ = isWrongType(expect, c.checkedValue, c.schema)
		return wrongType
	}
	return c.checkedValue[0].ID != gqlscan.TokenEnumVal
}

func (c *Checker) expectOrBool(expect *gqlast.Type) (wrongType bool) {
	if expect != nil {
		wrongType, _ = isWrongType(expect, c.checkedValue, c.schema)
		return wrongType
	}
	return c.checkedValue[0].ID != gqlscan.TokenTrue &&
		c.checkedValue[0].ID != gqlscan.TokenFalse
}

func (c *Checker) expectOrHasLen(
	expect *gqlast.Type,
) (wrongType bool) {
	if expect != nil {
		wrongType, _ = isWrongType(expect, c.checkedValue, c.schema)
		return wrongType
	}
	return c.checkedValue[0].ID != gqlscan.TokenArr &&
		c.checkedValue[0].ID != gqlscan.TokenStr &&
		c.checkedValue[0].ID != gqlscan.TokenStrBlock
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
	return func(c *Checker) (match bool) {
		if c.checkedValue[0].ID != gqlscan.TokenArr {
			return false
		}
		c.checkedValue = c.checkedValue[1:]
		count := 0
		for ; c.checkedValue[0].ID != gqlscan.TokenArrEnd; count++ {
			if count >= len(checks) {
				return false
			}
			if !checks[count](c) {
				return false
			}
		}
		if count != len(checks) {
			return false
		}
		c.checkedValue = c.checkedValue[1:]
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
	return func(c *Checker) (match bool) {
		if c.checkedValue[0].ID != gqlscan.TokenArr {
			return false
		}
		c.checkedValue = c.checkedValue[1:]
		count := 0
		for ; c.checkedValue[0].ID != gqlscan.TokenArrEnd; count++ {
			if count >= len(checks) {
				// Skip all following values to return correct tokenRead
				for c.checkedValue[0].ID != gqlscan.TokenArrEnd {
					l := gqlparse.GetValLen(c.checkedValue)
					c.checkedValue = c.checkedValue[l:]
				}
				return true
			}
			if !checks[count](c) {
				// Skip all following values to return correct tokenRead
				for c.checkedValue[0].ID != gqlscan.TokenArrEnd {
					l := gqlparse.GetValLen(c.checkedValue)
					c.checkedValue = c.checkedValue[l:]
				}
				return true
			}
		}
		c.checkedValue = c.checkedValue[1:]
		// All item checks were matched, finally check length
		return count != len(checks)
	}
}

func makeEqNumber(
	v *gqt.Number,
	expect *gqlast.Type,
	schema *gqlast.Schema,
) check {
	if i, ok := v.Int(); ok {
		return func(c *Checker) (match bool) {
			if c.expectOrInt(expect) {
				return false
			}
			a := atoi.MustI32(c.checkedValue[0].Value)
			c.checkedValue = c.checkedValue[1:]
			return int32(i) == a
		}
	}
	f, _ := v.Float()
	return func(c *Checker) (match bool) {
		if c.expectOrFloat(expect) {
			return false
		}
		a := atoi.MustF64(c.checkedValue[0].Value)
		c.checkedValue = c.checkedValue[1:]
		return f == a
	}
}

func makeNotEqNumber(
	v *gqt.Number,
	expect *gqlast.Type,
	schema *gqlast.Schema,
) check {
	if i, ok := v.Int(); ok {
		return func(c *Checker) (match bool) {
			if c.expectOrInt(expect) {
				return false
			}
			a := atoi.MustI32(c.checkedValue[0].Value)
			c.checkedValue = c.checkedValue[1:]
			return int32(i) != a
		}
	}
	f, _ := v.Float()
	return func(c *Checker) (match bool) {
		if c.expectOrFloat(expect) {
			return false
		}
		a := atoi.MustF64(c.checkedValue[0].Value)
		c.checkedValue = c.checkedValue[1:]
		return f != a
	}
}

func makeEqString(
	v *gqt.String,
	expect *gqlast.Type,
	schema *gqlast.Schema,
) check {
	return func(c *Checker) (match bool) {
		if c.expectOrString(expect) {
			return false
		}
		t := c.checkedValue[0]
		c.checkedValue = c.checkedValue[1:]
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
	return func(c *Checker) (match bool) {
		if c.expectOrString(expect) {
			return false
		}
		t := c.checkedValue[0]
		c.checkedValue = c.checkedValue[1:]
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
	return func(c *Checker) (match bool) {
		if c.expectOrEnum(expect) {
			return false
		}
		b := c.checkedValue[0].Value
		c.checkedValue = c.checkedValue[1:]
		return v.Value == string(b)
	}
}

func makeNotEqEnum(
	v *gqt.Enum,
	expect *gqlast.Type,
	schema *gqlast.Schema,
) check {
	return func(c *Checker) (match bool) {
		if c.expectOrEnum(expect) {
			return false
		}
		b := c.checkedValue[0].Value
		c.checkedValue = c.checkedValue[1:]
		return v.Value != string(b)
	}
}

func makeEqNull(
	expect *gqlast.Type,
	schema *gqlast.Schema,
) check {
	return func(c *Checker) (match bool) {
		if wrong, _ := isWrongType(expect, c.checkedValue, schema); wrong {
			return false
		}
		t := c.checkedValue[0].ID
		c.checkedValue = c.checkedValue[1:]
		return t == gqlscan.TokenNull
	}
}

func makeNotEqNull(
	expect *gqlast.Type,
	schema *gqlast.Schema,
) check {
	return func(c *Checker) (match bool) {
		if wrong, _ := isWrongType(expect, c.checkedValue, schema); wrong {
			return false
		}
		t := c.checkedValue[0].ID
		c.checkedValue = c.checkedValue[1:]
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
	return func(c *Checker) (match bool) {
		if c.expectOrBool(expect) {
			return false
		}
		t := c.checkedValue[0].ID
		c.checkedValue = c.checkedValue[1:]
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
	return func(c *Checker) (match bool) {
		if c.expectOrBool(expect) {
			return false
		}
		t := c.checkedValue[0].ID
		c.checkedValue = c.checkedValue[1:]
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
	return func(c *Checker) (match bool) {
		if wrong, _ := isWrongType(expect, c.checkedValue, schema); wrong {
			return false
		}
		if c.checkedValue[0].ID != gqlscan.TokenObj {
			return false
		}
		c.checkedValue = c.checkedValue[1:]

		// Reset check status
		for k := range fieldChecked {
			fieldChecked[k] = false
		}

		count := 0
		for ; c.checkedValue[0].ID != gqlscan.TokenObjEnd; count++ {
			if count >= len(checks) {
				return false
			}

			check := checks[string(c.checkedValue[0].Value)]
			if check == nil {
				// Unknown field, wrong object type
				return false
			}

			if fieldChecked[string(c.checkedValue[0].Value)] {
				// Field provided twice, invalid object
				return false
			}
			fieldChecked[string(c.checkedValue[0].Value)] = true

			c.checkedValue = c.checkedValue[1:]

			if !check(c) {
				return false
			}
		}
		if requiredChecks < 1 && count != len(checks) || count < requiredChecks {
			// Not all required fields were provided, wrong object type
			return false
		}
		c.checkedValue = c.checkedValue[1:]
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
	return func(c *Checker) (match bool) {
		if wrong, _ := isWrongType(expect, c.checkedValue, schema); wrong {
			return false
		}
		if c.checkedValue[0].ID != gqlscan.TokenObj {
			return false
		}
		c.checkedValue = c.checkedValue[1:]

		// Reset check status
		for k := range fieldChecked {
			fieldChecked[k] = false
		}

		count := 0
		for ; c.checkedValue[0].ID != gqlscan.TokenObjEnd; count++ {
			if count >= len(checks) {
				return false
			}

			check := checks[string(c.checkedValue[0].Value)]
			if check == nil {
				// Unknown field, wrong object type
				return false
			}

			if fieldChecked[string(c.checkedValue[0].Value)] {
				// Field provided twice, invalid object
				return false
			}
			fieldChecked[string(c.checkedValue[0].Value)] = true

			c.checkedValue = c.checkedValue[1:]

			if check(c) {
				// Don't return just yet as we're not sure whether the
				// object type was correct.
				match = true
			}
		}
		if requiredChecks < 1 && count != len(checks) || count < requiredChecks {
			// Not all required fields were provided, wrong object type
			return false
		}
		c.checkedValue = c.checkedValue[1:]
		return !match
	}
}

func makeEqExpression(
	v gqt.Expression,
	expect *gqlast.Type,
	schema *gqlast.Schema,
) check {
	return func(c *Checker) (match bool) {
		if wrong, _ := isWrongType(expect, c.checkedValue, schema); wrong {
			return false
		}
		c.resolveExpr(v)
		u := c.popStack()

		switch u.Type() {
		case union.TypeNull:
			return c.checkedValue[0].ID == gqlscan.TokenNull
		case union.TypeBoolean:
			b, _ := u.Bool()
			if b {
				return c.checkedValue[0].ID == gqlscan.TokenTrue
			}
			return c.checkedValue[0].ID == gqlscan.TokenFalse
		case union.TypeInt:
			if c.checkedValue[0].ID == gqlscan.TokenInt {
				u, _ := u.Int()
				return u == atoi.MustI32(c.checkedValue[0].Value)
			}
			return false
		case union.TypeFloat:
			if c.checkedValue[0].ID == gqlscan.TokenFloat {
				u, _ := u.Float()
				return u == atoi.MustF64(c.checkedValue[0].Value)
			}
			return false
		case union.TypeString:
			if c.checkedValue[0].ID == gqlscan.TokenStr {
				u, _ := u.String()
				return u == string(c.checkedValue[0].Value)
			}
			return false
		case union.TypeEnum:
			if c.checkedValue[0].ID == gqlscan.TokenEnumVal {
				u, _ := u.Enum()
				return u == string(c.checkedValue[0].Value)
			}
			return false
		case union.TypeTokens:
			if len(u.Tokens()) != len(c.checkedValue) {
				return false
			}
			for i := range c.checkedValue {
				if c.checkedValue[i].ID != u.Tokens()[i].ID {
					return false
				}
				switch c.checkedValue[i].ID {
				case gqlscan.TokenStrBlock:
					if string(c.checkedValue[i].Value) != string(u.Tokens()[i].Value) {
						return false
					}
				default:
					if string(c.checkedValue[i].Value) != string(u.Tokens()[i].Value) {
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
