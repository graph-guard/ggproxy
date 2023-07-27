// Package travgqt provides a GQT expression traversal function.
package travgqt

import (
	"fmt"

	"github.com/graph-guard/gqt/v4"
)

// Traverse returns true after BFS-traversing the entire tree under e
// calling onExpression for every discovered expression.
// Returns true immediatelly if onExpression returns stop=true.
func Traverse(
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
