package rmap

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"

	"github.com/graph-guard/gguard-proxy/engines/qmap"
	"github.com/graph-guard/gguard-proxy/gqlreduce"
	"github.com/graph-guard/gguard-proxy/matcher"
	"github.com/graph-guard/gguard-proxy/utilities/bitmask"
	"github.com/graph-guard/gguard-proxy/utilities/container/amap"
	"github.com/graph-guard/gguard-proxy/utilities/container/hamap"
	"github.com/graph-guard/gguard-proxy/utilities/xxhash"
	"github.com/graph-guard/gqt"
)

var ErrHashCollision = errors.New("hash collsision")

type ErrReducer struct {
	msg string
}

func (er *ErrReducer) Error() string {
	return er.msg
}

const (
	maxRand     = 32768
	maxAttempts = 32
)

// RulesMap is a graphql query to a template fast search structure.
type RulesMap struct {
	seed         uint64
	mask         *bitmask.Set
	qmake        qmap.Maker
	templates    [][]byte
	reducer      *gqlreduce.Reducer
	matchCounter *amap.Map[uint16, uint16]
	rules        map[uint64]*RulesNode
	ruleCounter  map[uint16]uint16
	hashedPaths  map[uint64]string
}

// RulesNode is an auxiliary RulesMap structure.
type RulesNode struct {
	Mask     *bitmask.Set
	Variants []Variant
}

// Variant is an auxiliary RulesNode structure.
type Variant struct {
	Condition  bool
	Constraint matcher.Constraint
	Mask       *bitmask.Set
	Value      any
}

// Elem is an auxiliary Variant structure.
type Elem struct {
	Constraint matcher.Constraint
	Value      any
}

// Array is an auxiliary array structure.
type Array []Elem

// Object is an auxiliary map structure.
type Object map[string]Elem

// ConstraintInterface is a generic interface for constraints.
type ConstraintInterface interface {
	Key() string
	Content() gqt.Constraint
	gqt.InputConstraint | gqt.ObjectField
}

// Equal checks two Elems for equality.
func (e Elem) Equal(x Elem) bool {
	if e.Constraint == x.Constraint {
		switch ve := e.Value.(type) {
		case Elem:
			switch vx := x.Value.(type) {
			case Elem:
				return ve.Equal(vx)
			}
		case Array:
			switch vx := x.Value.(type) {
			case Array:
				return ve.Equal(vx)
			}
		case Object:
			switch vx := x.Value.(type) {
			case Object:
				return ve.Equal(vx)
			}
		default:
			return e.Value == x.Value
		}
	}

	return false
}

// Equal checks two Arrays for equality.
func (arr Array) Equal(x Array) bool {
	if len(arr) != len(x) {
		return false
	}

	for i := 0; i < len(x); i++ {
		if !arr[i].Equal(x[i]) {
			return false
		}
	}

	return true
}

// Equal checks two Objects for equality.
func (obj Object) Equal(x Object) bool {
	if len(obj) != len(x) {
		return false
	}

	for k, e := range x {
		v, ok := obj[k]
		if !ok {
			return false
		}
		if !v.Equal(e) {
			return false
		}
	}

	return true
}

// New creates a new instance of RulesMap.
// Accepts a rules list and a hash seed.
func New(rules []gqt.Doc, seed uint64) (*RulesMap, error) {
	rm := &RulesMap{
		seed:         seed,
		mask:         bitmask.New(),
		qmake:        *qmap.NewMaker(seed),
		reducer:      gqlreduce.NewReducer(),
		matchCounter: amap.New[uint16, uint16](0),
		rules:        map[uint64]*RulesNode{},
		ruleCounter:  map[uint16]uint16{},
		hashedPaths:  map[uint64]string{},
	}
	var attempt int
	var err error

	for attempt < maxAttempts {
		for idx, rule := range rules {
			m := bitmask.New(idx)
			switch r := rule.(type) {
			case gqt.DocQuery:
				err = buildRulesMapSelections(rm, r.Selections, m, "query", uint16(idx))
			case gqt.DocMutation:
				err = buildRulesMapSelections(rm, r.Selections, m, "mutation", uint16(idx))
			}
			if err == ErrHashCollision {
				rm.seed = uint64(rand.Int31n(maxRand))
				rm = &RulesMap{
					rules:        map[uint64]*RulesNode{},
					ruleCounter:  map[uint16]uint16{},
					mask:         bitmask.New(),
					hashedPaths:  map[uint64]string{},
					matchCounter: amap.New[uint16, uint16](0),
					qmake:        *qmap.NewMaker(rm.seed),
				}
				attempt++
				break
			}
		}
		if err != ErrHashCollision {
			break
		}
	}

	if err != nil {
		return nil, err
	}

	return rm, nil
}

func buildRulesMapSelections(
	rm *RulesMap,
	selections []gqt.Selection,
	mask *bitmask.Set,
	path string,
	ruleIdx uint16,
) error {
	for _, selection := range selections {
		switch selection := selection.(type) {
		case gqt.SelectionField:
			selPath := path + "." + selection.Name
			if len(selection.Selections) == 0 && len(selection.InputConstraints) == 0 {
				h := xxhash.New(rm.seed)
				xxhash.Write(&h, selPath)
				pathHash := h.Sum64()
				if v, ok := (*rm).rules[pathHash]; ok {
					v.Mask = v.Mask.Or(mask)
				} else {
					if v, ok := rm.hashedPaths[pathHash]; !ok {
						rm.hashedPaths[pathHash] = selPath
					} else {
						if v != selPath {
							return ErrHashCollision
						}
					}
					(*rm).rules[pathHash] = &RulesNode{
						Mask: mask,
					}
				}
				(*rm).ruleCounter[ruleIdx]++
			} else {
				if len(selection.Selections) > 0 {
					if err := buildRulesMapSelections(
						rm, selection.Selections, mask, selPath, ruleIdx,
					); err != nil {
						return err
					}
				}
				if len(selection.InputConstraints) > 0 {
					if err := buildRulesMapConstraints(
						rm, selection.InputConstraints, mask, selPath, true, ruleIdx,
					); err != nil {
						return err
					}
				}
			}
		case gqt.SelectionInlineFragment:
			panic("can't work with fragments yet")
		}
	}

	return nil
}

func buildRulesMapConstraints[T ConstraintInterface](
	rm *RulesMap,
	constraints []T,
	mask *bitmask.Set,
	path string,
	condition bool,
	ruleIdx uint16,
) error {
	for _, constraint := range constraints {
		var cv any
		var cid matcher.Constraint
		cond := condition

		cid, cv = ConstraintIdAndValue(constraint.Content())
		if cid == matcher.ConstraintValNotEqual {
			cond = !cond
		}

		conPath := path + "." + constraint.Key()
		switch cv := cv.(type) {
		case gqt.ValueObject:
			if err := buildRulesMapConstraints(rm, cv.Fields, mask, conPath, cond, ruleIdx); err != nil {
				return err
			}
		default:
			h := xxhash.New(rm.seed)
			xxhash.Write(&h, conPath)
			pathHash := h.Sum64()
			if v, ok := (*rm).rules[pathHash]; ok {
				v.Mask = v.Mask.Or(mask)
			} else {
				if v, ok := rm.hashedPaths[pathHash]; !ok {
					rm.hashedPaths[pathHash] = conPath
				} else {
					if v != conPath {
						return ErrHashCollision
					}
				}
				(*rm).rules[pathHash] = &RulesNode{
					Mask: mask,
				}
			}
			(*rm).ruleCounter[ruleIdx]++
			switch cid {
			case matcher.ConstraintMap:
				(*rm).rules[pathHash].Variants = mergeVariants((*rm).rules[pathHash].Variants, Variant{
					Condition:  cond,
					Constraint: cid,
					Mask:       mask,
					Value:      buildRulesMapConstraintsElem(cv),
				})
			case matcher.ConstraintOr, matcher.ConstraintAnd:
				(*rm).rules[pathHash].Variants = mergeVariants((*rm).rules[pathHash].Variants, Variant{
					Condition:  cond,
					Constraint: cid,
					Mask:       mask,
					Value:      buildRulesMapConstraintsArray(cv.([]gqt.Constraint)),
				})
			default:
				switch cv := cv.(type) {
				case gqt.ValueArray:
					(*rm).rules[pathHash].Variants = mergeVariants((*rm).rules[pathHash].Variants, Variant{
						Condition:  cond,
						Constraint: cid,
						Mask:       mask,
						Value:      buildRulesMapConstraintsArray(cv.Items),
					})
				default:
					(*rm).rules[pathHash].Variants = mergeVariants((*rm).rules[pathHash].Variants, Variant{
						Condition:  cond,
						Constraint: cid,
						Mask:       mask,
						Value:      cv,
					})
				}
			}
		}
	}

	return nil
}

func buildRulesMapConstraintsElem(constraint gqt.Constraint) (el Elem) {
	var cv any
	var cid matcher.Constraint

	cid, cv = ConstraintIdAndValue(constraint)

	switch cid {
	case matcher.ConstraintMap:
		el = Elem{
			Constraint: cid,
			Value:      buildRulesMapConstraintsElem(cv),
		}
	case matcher.ConstraintOr, matcher.ConstraintAnd:
		el = Elem{
			Constraint: cid,
			Value:      buildRulesMapConstraintsArray(cv.([]gqt.Constraint)),
		}
	default:
		switch cv := cv.(type) {
		case gqt.ValueObject:
			el = Elem{
				Constraint: cid,
				Value:      buildRulesMapConstraintsObject(cv.Fields),
			}
		case gqt.ValueArray:
			el = Elem{
				Constraint: cid,
				Value:      buildRulesMapConstraintsArray(cv.Items),
			}
		default:
			el = Elem{
				Constraint: cid,
				Value:      cv,
			}
		}
	}

	return
}

func buildRulesMapConstraintsArray(
	constraints []gqt.Constraint,
) (arr Array) {
	for _, constraint := range constraints {
		arr = append(arr, buildRulesMapConstraintsElem(constraint))
	}

	return
}

func buildRulesMapConstraintsObject(
	constraints []gqt.ObjectField,
) (obj Object) {
	obj = map[string]Elem{}
	for _, constraint := range constraints {
		obj[constraint.Name] = buildRulesMapConstraintsElem(constraint.Value)
	}

	return
}

func mergeVariants(variants []Variant, x Variant) []Variant {
	for i := 0; i < len(variants); i++ {
		if variants[i].Constraint == x.Constraint {
			switch vt := variants[i].Value.(type) {
			case Elem:
				switch xt := x.Value.(type) {
				case Elem:
					if vt.Equal(xt) {
						variants[i].Mask = variants[i].Mask.Or(x.Mask)
						return variants
					}
				}
			case Array:
				switch xt := x.Value.(type) {
				case Array:
					if vt.Equal(xt) {
						variants[i].Mask = variants[i].Mask.Or(x.Mask)
						return variants
					}
				}
			case Object:
				switch xt := x.Value.(type) {
				case Object:
					if vt.Equal(xt) {
						variants[i].Mask = variants[i].Mask.Or(x.Mask)
						return variants
					}
				}
			default:
				if xb, ok := x.Value.([]byte); ok {
					if bytes.Equal(variants[i].Value.([]byte), xb) {
						variants[i].Mask = variants[i].Mask.Or(x.Mask)
						return variants
					}
				} else if variants[i].Value == x.Value {
					variants[i].Mask = variants[i].Mask.Or(x.Mask)
					return variants
				}
			}
		}
	}
	variants = append(variants, x)

	return variants
}

// ConstraintIdAndValue returns constraint Id and Value.
func ConstraintIdAndValue(c gqt.Constraint) (matcher.Constraint, any) {
	switch c := c.(type) {
	case gqt.ConstraintOr:
		return matcher.ConstraintOr, c.Constraints
	case gqt.ConstraintAnd:
		return matcher.ConstraintAnd, c.Constraints
	case gqt.ConstraintMap:
		return matcher.ConstraintMap, c.Constraint
	case gqt.ConstraintAny:
		return matcher.ConstraintAny, nil
	case gqt.ConstraintValEqual:
		if s, ok := c.Value.(string); ok {
			return matcher.ConstraintValEqual, []byte(s)
		}
		return matcher.ConstraintValEqual, c.Value
	case gqt.ConstraintValNotEqual:
		if s, ok := c.Value.(string); ok {
			return matcher.ConstraintValNotEqual, []byte(s)
		}
		return matcher.ConstraintValNotEqual, c.Value
	case gqt.ConstraintValGreater:
		return matcher.ConstraintValGreater, c.Value
	case gqt.ConstraintValLess:
		return matcher.ConstraintValLess, c.Value
	case gqt.ConstraintValGreaterOrEqual:
		return matcher.ConstraintValGreaterOrEqual, c.Value
	case gqt.ConstraintValLessOrEqual:
		return matcher.ConstraintValLessOrEqual, c.Value
	case gqt.ConstraintBytelenEqual:
		return matcher.ConstraintBytelenEqual, c.Value
	case gqt.ConstraintBytelenNotEqual:
		return matcher.ConstraintBytelenNotEqual, c.Value
	case gqt.ConstraintBytelenGreater:
		return matcher.ConstraintBytelenGreater, c.Value
	case gqt.ConstraintBytelenLess:
		return matcher.ConstraintBytelenLess, c.Value
	case gqt.ConstraintBytelenGreaterOrEqual:
		return matcher.ConstraintBytelenGreaterOrEqual, c.Value
	case gqt.ConstraintBytelenLessOrEqual:
		return matcher.ConstraintBytelenLessOrEqual, c.Value
	case gqt.ConstraintLenEqual:
		return matcher.ConstraintLenEqual, c.Value
	case gqt.ConstraintLenNotEqual:
		return matcher.ConstraintLenNotEqual, c.Value
	case gqt.ConstraintLenGreater:
		return matcher.ConstraintLenGreater, c.Value
	case gqt.ConstraintLenLess:
		return matcher.ConstraintLenLess, c.Value
	case gqt.ConstraintLenGreaterOrEqual:
		return matcher.ConstraintLenGreaterOrEqual, c.Value
	case gqt.ConstraintLenLessOrEqual:
		return matcher.ConstraintLenLessOrEqual, c.Value
	default:
		return matcher.ConstraintUnknown, nil
	}
}

// Match searches for a matching template.
// Accepts a text query, query variables and operation name.
// Returns nil if matching template is found, ErrNoMatch if not found.
func (rm *RulesMap) Match(
	ctx context.Context,
	query []byte,
	operationName []byte,
	variablesJSON []byte,
) (err error) {
	var match bool
	rm.reducer.Reduce(
		query, operationName, variablesJSON,
		func(operation []gqlreduce.Token) {
			rm.FindMatch(operation, func(mask *bitmask.Set) {
				if mask.Size() > 0 {
					match = true
				}
			})
		},
		func(e error) {
			err = &ErrReducer{msg: fmt.Sprintf("reducer error: %s", e.Error())}
		},
	)
	if err != nil {
		return err
	}
	if !match {
		return matcher.ErrNoMatch
	}
	return nil
}

// MatchAll searches for a matching templates.
// Accepts a text query, query variables, operation name and function that gets a matched mask as an input.
func (rm *RulesMap) MatchAll(
	ctx context.Context,
	query []byte,
	operationName []byte,
	variablesJSON []byte,
	fn func(idx int),
) (err error) {
	rm.reducer.Reduce(
		query, operationName, variablesJSON,
		func(operation []gqlreduce.Token) {
			rm.FindMatch(operation, func(mask *bitmask.Set) {
				mask.Visit(func(n int) (skip bool) {
					fn(n)
					return false
				})
			})
		},
		func(e error) {
			err = &ErrReducer{msg: fmt.Sprintf("reducer error: %s", e.Error())}
		},
	)
	return err
}

// GetTemplate returns a template at the index or and error if no such index.
func (rm *RulesMap) GetTemplate(ctx context.Context, index uint16) ([]byte, error) {
	if index < uint16(len(rm.templates)) {
		return rm.templates[index], nil
	}

	return nil, matcher.ErrTemplateNotFound
}

// VisitTemplates loops through the preserved templates and calls a function on every template.
func (rm *RulesMap) VisitTemplates(ctx context.Context, fn func(template []byte) (stop bool)) {
	for _, t := range rm.templates {
		if fn(t) {
			break
		}
	}
}

// FindMatch matches query to the rules.
func (rm *RulesMap) FindMatch(query []gqlreduce.Token, fn func(mask *bitmask.Set)) {
	rm.qmake.ParseQuery(query, func(qm qmap.QueryMap) {
		rm.matchCounter.Reset()
		rm.mask.Reset()

		for hash, value := range qm {
			if rn, ok := rm.rules[hash]; ok {
				if len(rn.Variants) > 0 {
					for _, v := range rn.Variants {
						if v.Compare(value) {
							rm.mask.SetOr(rm.mask, v.Mask)
							v.Mask.Visit(func(x int) (skip bool) {
								rm.matchCounter.SetFn(uint16(x), 1, func(value *uint16) { *value++ })
								return false
							})
						}
					}
				} else {
					rm.mask.SetOr(rm.mask, rn.Mask)
					rn.Mask.Visit(func(x int) (skip bool) {
						rm.matchCounter.SetFn(uint16(x), 1, func(value *uint16) { *value++ })
						return false
					})
				}
			} else {
				rm.mask.Reset()
				break
			}
		}
		for _, el := range rm.matchCounter.A {
			if el.Value < uint16(len(qm)) || el.Value != rm.ruleCounter[el.Key] {
				rm.mask = rm.mask.Delete(int(el.Key))
			}
		}

		if rm.mask.Empty() {
			rm.mask.Reset()
		}

		fn(rm.mask)
	})
}

// CompareValues compares two values according to the provided constraint.
func CompareValues(constraint matcher.Constraint, a any, b any) bool {
	switch constraint {
	case matcher.ConstraintAny:
		return true
	case matcher.ConstraintValEqual:
		if b, ok := b.([]byte); ok {
			return bytes.Equal(b, a.([]byte))
		}
		return b == a
	case matcher.ConstraintValNotEqual:
		if b, ok := b.([]byte); ok {
			return !bytes.Equal(b, a.([]byte))
		}
		return b != a
	case matcher.ConstraintValGreater, matcher.ConstraintValLess,
		matcher.ConstraintValGreaterOrEqual, matcher.ConstraintValLessOrEqual:
		switch vala := a.(type) {
		case int64:
			valb, ok := b.(int64)
			if !ok {
				return false
			}
			switch constraint {
			case matcher.ConstraintValGreater:
				return valb > vala
			case matcher.ConstraintValLess:
				return valb < vala
			case matcher.ConstraintValGreaterOrEqual:
				return valb >= vala
			case matcher.ConstraintValLessOrEqual:
				return valb <= vala
			}
		case float64:
			valb, ok := b.(float64)
			if !ok {
				return false
			}
			switch constraint {
			case matcher.ConstraintValGreater:
				return valb > vala
			case matcher.ConstraintValLess:
				return valb < vala
			case matcher.ConstraintValGreaterOrEqual:
				return valb >= vala
			case matcher.ConstraintValLessOrEqual:
				return valb <= vala
			}
		}
	case matcher.ConstraintBytelenEqual, matcher.ConstraintBytelenNotEqual,
		matcher.ConstraintBytelenGreater, matcher.ConstraintBytelenLess,
		matcher.ConstraintBytelenGreaterOrEqual, matcher.ConstraintBytelenLessOrEqual:
		vala, ok := a.(uint)
		if !ok {
			return false
		}
		valb, ok := b.([]byte)
		if !ok {
			return false
		}
		switch constraint {
		case matcher.ConstraintBytelenEqual:
			return len(valb) == int(vala)
		case matcher.ConstraintBytelenNotEqual:
			return len(valb) != int(vala)
		case matcher.ConstraintBytelenGreater:
			return len(valb) > int(vala)
		case matcher.ConstraintBytelenLess:
			return len(valb) < int(vala)
		case matcher.ConstraintBytelenGreaterOrEqual:
			return len(valb) >= int(vala)
		case matcher.ConstraintBytelenLessOrEqual:
			return len(valb) <= int(vala)
		}
	case matcher.ConstraintLenEqual, matcher.ConstraintLenNotEqual,
		matcher.ConstraintLenGreater, matcher.ConstraintLenLess,
		matcher.ConstraintLenGreaterOrEqual, matcher.ConstraintLenLessOrEqual:
		ca, ok := a.(uint)
		if !ok {
			return false
		}
		bi, ok := b.(*[]any)
		if !ok {
			return false
		}
		switch constraint {
		case matcher.ConstraintLenEqual:
			return len(*bi) == int(ca)
		case matcher.ConstraintLenNotEqual:
			return len(*bi) != int(ca)
		case matcher.ConstraintLenGreater:
			return len(*bi) > int(ca)
		case matcher.ConstraintLenLess:
			return len(*bi) < int(ca)
		case matcher.ConstraintLenGreaterOrEqual:
			return len(*bi) >= int(ca)
		case matcher.ConstraintLenLessOrEqual:
			return len(*bi) <= int(ca)
		}
	default:
		panic(fmt.Errorf("wrong constraint-type pair; constraint: %d", constraint))
	}

	return true
}

// Compare checks two Varians for equality.
func (v Variant) Compare(x any) bool {
	switch v.Constraint {
	case matcher.ConstraintMap:
		switch xt := x.(type) {
		case *[]any:
			for _, el := range *xt {
				switch vt := v.Value.(type) {
				case Elem:
					if !vt.Compare(el) {
						return false
					}
				case Array:
					switch elt := el.(type) {
					case *[]any:
						if !vt.Compare(*elt) {
							return false
						}
					}
				case Object:
					switch elt := el.(type) {
					case *hamap.Map[string, any]:
						if !vt.Compare(elt) {
							return false
						}
					}
				default:
					if !CompareValues(v.Constraint, v.Value, x) {
						return false
					}
				}
			}
			return true
		}
	case matcher.ConstraintOr:
		for _, el := range v.Value.(Array) {
			if el.Compare(x) {
				return true
			}
		}
		return false
	case matcher.ConstraintAnd:
		for _, el := range v.Value.(Array) {
			if !el.Compare(x) {
				return false
			}
		}
		return true
	default:
		neq := v.Constraint == matcher.ConstraintValNotEqual
		switch vt := v.Value.(type) {
		case Elem:
			return vt.Compare(x) != neq
		case Array:
			switch xt := x.(type) {
			case *[]any:
				return vt.Compare(*xt) != neq
			}
		case Object:
			switch xt := x.(type) {
			case *hamap.Map[string, any]:
				return vt.Compare(xt) != neq
			}
		default:
			return CompareValues(v.Constraint, v.Value, x)
		}
	}

	return false
}

// Compare checks two Elems for equality.
func (e Elem) Compare(x any) bool {
	switch e.Constraint {
	case matcher.ConstraintMap:
		switch xt := x.(type) {
		case *[]any:
			for _, el := range *xt {
				switch et := e.Value.(type) {
				case Elem:
					return et.Compare(el)
				case Array:
					switch elt := el.(type) {
					case *[]any:
						return et.Compare(*elt)
					}
				case Object:
					switch elt := el.(type) {
					case *hamap.Map[string, any]:
						return et.Compare(elt)
					}
				default:
					return CompareValues(e.Constraint, e.Value, x)
				}
			}
		}
	case matcher.ConstraintOr:
		for _, el := range e.Value.(Array) {
			if el.Compare(x) {
				return true
			}
		}
		return false
	case matcher.ConstraintAnd:
		for _, el := range e.Value.(Array) {
			if !el.Compare(x) {
				return false
			}
		}
		return true
	default:
		neq := e.Constraint == matcher.ConstraintValNotEqual
		switch et := e.Value.(type) {
		case Elem:
			return et.Compare(x) != neq
		case Array:
			switch xt := x.(type) {
			case *[]any:
				return et.Compare(*xt) != neq
			}
		case Object:
			switch xt := x.(type) {
			case *hamap.Map[string, any]:
				return et.Compare(xt) != neq
			}
		default:
			return CompareValues(e.Constraint, e.Value, x)
		}
	}

	return false
}

// Compare checks two Arrays for equality.
func (arr Array) Compare(x []any) bool {
	if len(arr) != len(x) {
		return false
	}
	for i := 0; i < len(x); i++ {
		if !arr[i].Compare(x[i]) {
			return false
		}
	}

	return true
}

// Compare checks two Objects for equality.
func (obj Object) Compare(x *hamap.Map[string, any]) (eq bool) {
	var v Elem
	eq = true
	if len(obj) != x.Len() {
		return false
	}
	x.Visit(func(key string, value any) (stop bool) {
		v, eq = obj[key]
		if !eq {
			return true
		}
		eq = v.Compare(value)
		return !eq
	})

	return
}

// PrintNSpaces prints n spaces in a row.
func PrintNSpaces(w io.Writer, n uint) {
	for i := uint(0); i < n; i++ {
		_, _ = w.Write([]byte(" "))
	}
}

// Print prints out the RulesMap object.
func (rm *RulesMap) Print(w io.Writer) {
	rm.print(w, 0)
}

func (rm *RulesMap) print(w io.Writer, indent uint) {
	for hash, rn := range rm.rules {
		fmt.Fprintf(w, "%d", hash)
		_, _ = w.Write([]byte(": "))
		rn.Mask.Visit(func(x int) (skip bool) {
			fmt.Fprintf(w, "%d", x)
			return false
		})
		_, _ = w.Write([]byte("\n"))
		if len(rn.Variants) > 0 {
			PrintNSpaces(w, indent+2)
			_, _ = w.Write([]byte("variants:\n"))
			for _, v := range rn.Variants {
				v.print(w, indent+4)
			}
		}
	}
}

func (v *Variant) print(w io.Writer, indent uint) {
	PrintNSpaces(w, indent)
	_, _ = w.Write(append([]byte(matcher.ConstraintLookup[v.Constraint]), []byte(": ")...))
	v.Mask.Visit(func(x int) (skip bool) {
		fmt.Fprintf(w, "%d", x)
		return false
	})
	_, _ = w.Write([]byte("\n"))
	if v.Value != nil {
		switch v := v.Value.(type) {
		case Elem:
			v.print(w, indent+2)
		case Array:
			v.print(w, indent+2)
		default:
			PrintNSpaces(w, indent+2)
			if s, ok := v.([]byte); ok {
				fmt.Fprintln(w, string(s))
			} else {
				fmt.Fprintln(w, v)
			}
		}
	}
}

func (e *Elem) print(w io.Writer, indent uint) {
	PrintNSpaces(w, indent)
	_, _ = w.Write(append([]byte(matcher.ConstraintLookup[e.Constraint]), []byte(":\n")...))
	switch v := e.Value.(type) {
	case Elem:
		v.print(w, indent+2)
	case Array:
		v.print(w, indent+2)
	case Object:
		v.print(w, indent+2)
	default:
		PrintNSpaces(w, indent+2)
		if s, ok := v.([]byte); ok {
			fmt.Fprintln(w, string(s))
		} else {
			fmt.Fprintln(w, v)
		}
	}
}

func (arr *Array) print(w io.Writer, indent uint) {
	for _, el := range *arr {
		PrintNSpaces(w, indent)
		_, _ = w.Write([]byte("-:\n"))
		PrintNSpaces(w, indent+2)
		_, _ = w.Write(append([]byte(matcher.ConstraintLookup[el.Constraint]), []byte(":\n")...))
		switch v := el.Value.(type) {
		case Elem:
			v.print(w, indent+4)
		case Array:
			v.print(w, indent+4)
		case Object:
			v.print(w, indent+4)
		default:
			PrintNSpaces(w, indent+4)
			if s, ok := v.([]byte); ok {
				fmt.Fprintln(w, string(s))
			} else {
				fmt.Fprintln(w, v)
			}
		}
	}
}

func (obj *Object) print(w io.Writer, indent uint) {
	for k, el := range *obj {
		PrintNSpaces(w, indent)
		_, _ = w.Write(append([]byte(k), []byte(":\n")...))
		PrintNSpaces(w, indent+2)
		_, _ = w.Write(append([]byte(matcher.ConstraintLookup[el.Constraint]), []byte(":\n")...))
		switch v := el.Value.(type) {
		case Elem:
			v.print(w, indent+4)
		case Array:
			v.print(w, indent+4)
		case Object:
			v.print(w, indent+4)
		default:
			PrintNSpaces(w, indent+4)
			if s, ok := v.([]byte); ok {
				fmt.Fprintln(w, string(s))
			} else {
				fmt.Fprintln(w, v)
			}
		}
	}
}
