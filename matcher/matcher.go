package matcher

import (
	"context"
	"errors"
)

// Constraint is a constraint simplified abstraction.
type Constraint uint16

const (
	ConstraintUnknown Constraint = iota
	ConstraintOr
	ConstraintAnd
	ConstraintAny
	ConstraintMap
	ConstraintTypeEqual
	ConstraintTypeNotEqual
	ConstraintValEqual
	ConstraintValNotEqual
	ConstraintValGreater
	ConstraintValLess
	ConstraintValGreaterOrEqual
	ConstraintValLessOrEqual
	ConstraintBytelenEqual
	ConstraintBytelenNotEqual
	ConstraintBytelenGreater
	ConstraintBytelenLess
	ConstraintBytelenGreaterOrEqual
	ConstraintBytelenLessOrEqual
	ConstraintLenEqual
	ConstraintLenNotEqual
	ConstraintLenGreater
	ConstraintLenLess
	ConstraintLenGreaterOrEqual
	ConstraintLenLessOrEqual
)

var ConstraintLookup = map[Constraint]string{
	ConstraintOr:                    "ConstraintOr",
	ConstraintAnd:                   "ConstraintAnd",
	ConstraintAny:                   "ConstraintAny",
	ConstraintMap:                   "ConstraintMap",
	ConstraintTypeEqual:             "ConstraintTypeEqual",
	ConstraintTypeNotEqual:          "ConstraintTypeNotEqual",
	ConstraintValEqual:              "ConstraintValEqual",
	ConstraintValNotEqual:           "ConstraintValNotEqual",
	ConstraintValGreater:            "ConstraintValGreater",
	ConstraintValLess:               "ConstraintValLess",
	ConstraintValGreaterOrEqual:     "ConstraintValGreaterOrEqual",
	ConstraintValLessOrEqual:        "ConstraintValLessOrEqual",
	ConstraintBytelenEqual:          "ConstraintBytelenEqual",
	ConstraintBytelenNotEqual:       "ConstraintBytelenNotEqual",
	ConstraintBytelenGreater:        "ConstraintBytelenGreater",
	ConstraintBytelenLess:           "ConstraintBytelenLess",
	ConstraintBytelenGreaterOrEqual: "ConstraintBytelenGreaterOrEqual",
	ConstraintBytelenLessOrEqual:    "ConstraintBytelenLessOrEqual",
	ConstraintLenEqual:              "ConstraintLenEqual",
	ConstraintLenNotEqual:           "ConstraintLenNotEqual",
	ConstraintLenGreater:            "ConstraintLenGreater",
	ConstraintLenLess:               "ConstraintLenLess",
	ConstraintLenGreaterOrEqual:     "ConstraintLenGreaterOrEqual",
	ConstraintLenLessOrEqual:        "ConstraintLenLessOrEqual",
}

var ErrNoMatch = errors.New("no match")
var ErrSyntax = errors.New("syntax error")
var ErrTemplateNotFound = errors.New("template not found")
var ErrOpUndefined = errors.New("operation undefined")

// Matcher is a generic matcher interface.
type Matcher interface {
	// Match returns nil if query matches any template,
	// otherwise returns ErrNoMatch.
	// Returns ErrSyntax in case of a syntax error in query.
	// Returns ErrOpUndefined if operationName != "" and
	// the operation isn't defined in query.
	Match(
		ctx context.Context,
		query, operationName, variablesJSON []byte,
	) error

	// MatchDebug calls fn for each template that matches query,
	// otherwise returns ErrNoMatch.
	// Returns ErrSyntax in case of a syntax error in query.
	// Returns ErrOpUndefined if operationName != "" and
	// the operation isn't defined in query.
	MatchAll(
		ctx context.Context,
		query, operationName, variablesJSON []byte,
		fn func(templateIndex int),
	) error

	// GetTemplate returns the template for the given index.
	// Returns nil if no template was found.
	GetTemplate(
		ctx context.Context,
		index uint16,
	) ([]byte, error)

	// VisitTemplates calls fn for every template in the acceptlist.
	VisitTemplates(
		ctx context.Context,
		fn func(template []byte) (stop bool),
	)
}
