package rmap

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"github.com/graph-guard/ggproxy/engines/rmap/pquery"
	"github.com/graph-guard/ggproxy/gqlparse"
	"github.com/graph-guard/ggproxy/utilities/bitmask"
	"github.com/graph-guard/ggproxy/utilities/container/amap"
	"github.com/graph-guard/ggproxy/utilities/container/hamap"
	"github.com/graph-guard/ggproxy/utilities/xxhash"
	"github.com/graph-guard/gqlscan"
	"github.com/graph-guard/gqt"
)

var ErrHashCollision = errors.New("hash collsision")

const (
	maxRand     = 32768
	maxAttempts = 32
)

// RulesMap is a graphql query to a template fast search structure.
type RulesMap struct {
	seed                uint64
	mask                *bitmask.Set
	rejected            *bitmask.Set
	qmake               pquery.Maker
	matchCounter        *amap.Map[int, int]
	combinations        []int
	combinationCounters []int
	rules               map[uint64][]Variant
	hashedPaths         map[uint64]string
	templateIDs         []string
}

// Combination is a auxiliary "max" block structure.
type Combination struct {
	Index     int
	Depth     int
	RuleIndex int
}

// Variant is an auxiliary RulesNode structure.
type Variant struct {
	Condition    bool
	Constraint   Constraint
	Mask         *bitmask.Set
	Value        any
	Combinations []Combination
}

// Elem is an auxiliary Variant structure.
type Elem struct {
	Constraint Constraint
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
		case []byte:
			switch vx := x.Value.(type) {
			case []byte:
				return bytes.Equal(ve, vx)
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
func New(rules map[string]gqt.Doc, seed uint64) (*RulesMap, error) {
	rm := &RulesMap{
		seed:                seed,
		mask:                bitmask.New(),
		rejected:            bitmask.New(),
		qmake:               *pquery.NewMaker(seed),
		matchCounter:        amap.New[int, int](0),
		combinations:        []int{},
		combinationCounters: []int{},
		rules:               map[uint64][]Variant{},
		hashedPaths:         map[uint64]string{},
	}
	var attempt int
	var err error

	rm.templateIDs = make([]string, 0, len(rules))
	for id := range rules {
		rm.templateIDs = append(rm.templateIDs, id)
	}

	for attempt < maxAttempts {
		for index, id := range rm.templateIDs {
			rule := rules[id]
			m := bitmask.New(index)
			if rule.Query != nil {
				err = buildRulesMapSelections(
					rm, rule.Query, nil, m, "query", index, 0,
				)
			}
			if rule.Mutation != nil {
				err = buildRulesMapSelections(
					rm, rule.Mutation, nil, m, "mutation", index, 0,
				)
			}
			if rule.Subscription != nil {
				panic("subscriptions are not yet supported")
			}
			if err == ErrHashCollision {
				rm = &RulesMap{
					seed:                seed,
					mask:                bitmask.New(),
					rejected:            bitmask.New(),
					qmake:               *pquery.NewMaker(seed),
					matchCounter:        amap.New[int, int](0),
					combinations:        []int{},
					combinationCounters: []int{},
					rules:               map[uint64][]Variant{},
					hashedPaths:         map[uint64]string{},
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
	dependencies []uint64,
	mask *bitmask.Set,
	path string,
	ruleIdx int,
	combinationDepth int,
) error {
	for _, selection := range selections {
		switch selection := selection.(type) {
		case gqt.SelectionField:
			selPath := path + "." + selection.Name
			if len(selection.Selections) == 0 && len(selection.InputConstraints) == 0 {
				h := xxhash.New(rm.seed)
				xxhash.Write(&h, selPath)
				pathHash := h.Sum64()

				v := Variant{
					Mask:         mask,
					Combinations: []Combination{},
				}
				if combinationDepth > 0 {
					v.Combinations = append(
						v.Combinations,
						Combination{len(rm.combinations) - 1, combinationDepth - 1, ruleIdx},
					)
				}
				if _, ok := (*rm).rules[pathHash]; ok {
					(*rm).rules[pathHash] = mergeVariants((*rm).rules[pathHash], v)
				} else {
					if v, ok := rm.hashedPaths[pathHash]; !ok {
						rm.hashedPaths[pathHash] = selPath
					} else {
						if v != selPath {
							return ErrHashCollision
						}
					}
					(*rm).rules[pathHash] = []Variant{v}
				}
			} else {
				var leafs []uint64
				var err error
				if len(selection.InputConstraints) > 0 {
					leafs, err = buildRulesMapConstraints(
						rm, selection.InputConstraints, dependencies, mask, selPath, true, ruleIdx, combinationDepth,
					)
					if err != nil {
						return err
					}
				}
				if len(selection.Selections) > 0 {
					if len(selection.InputConstraints) > 0 {
						combinationDepth = 0
					}
					err = buildRulesMapSelections(
						rm, selection.Selections, append(leafs, dependencies...), mask, selPath, ruleIdx, combinationDepth,
					)
					if err != nil {
						return err
					}
				}
			}
		case gqt.SelectionInlineFragment:
			selPath := path + ".|" + selection.TypeName
			if err := buildRulesMapSelections(
				rm, selection.Selections, dependencies, mask, selPath, ruleIdx, combinationDepth,
			); err != nil {
				return err
			}
		case gqt.ConstraintCombine:
			rm.combinations = append(rm.combinations, int(selection.MaxItems))
			rm.combinationCounters = append(rm.combinationCounters, 0)
			if err := buildRulesMapSelections(
				rm, selection.Items, dependencies, mask, path, ruleIdx, combinationDepth+1,
			); err != nil {
				return err
			}
		}
	}

	return nil
}

func buildRulesMapConstraints[T ConstraintInterface](
	rm *RulesMap,
	constraints []T,
	dependencies []uint64,
	mask *bitmask.Set,
	path string,
	condition bool,
	ruleIdx int,
	combinationDepth int,
) ([]uint64, error) {
	var leafs []uint64
	for _, constraint := range constraints {
		var cv any
		var cid Constraint
		cond := condition

		cid, cv = ConstraintIdAndValue(constraint.Content())
		if cid == ConstraintValNotEqual {
			cond = !cond
		}

		conPath := path + "." + constraint.Key()
		switch cv := cv.(type) {
		case gqt.ValueObject:
			l, err := buildRulesMapConstraints(
				rm, cv.Fields, dependencies, mask, conPath, cond, ruleIdx, combinationDepth,
			)
			if err != nil {
				return nil, err
			}
			leafs = append(leafs, l...)
		default:
			h := xxhash.New(rm.seed)
			xxhash.Write(&h, conPath)
			pathHash := h.Sum64()
			leafs = append(leafs, pathHash)
			if _, ok := (*rm).rules[pathHash]; !ok {
				if v, ok := rm.hashedPaths[pathHash]; !ok {
					rm.hashedPaths[pathHash] = conPath
				} else {
					if v != conPath {
						return nil, ErrHashCollision
					}
				}
			}

			c := []Combination{}
			if combinationDepth > 0 {
				c = append(
					c,
					Combination{len(rm.combinations) - 1, combinationDepth - 1, ruleIdx},
				)
			}
			switch cid {
			case ConstraintMap:
				(*rm).rules[pathHash] = mergeVariants((*rm).rules[pathHash], Variant{
					Condition:    cond,
					Constraint:   cid,
					Mask:         mask,
					Value:        buildRulesMapConstraintsElem(cv),
					Combinations: c,
				})
			case ConstraintOr, ConstraintAnd:
				(*rm).rules[pathHash] = mergeVariants((*rm).rules[pathHash], Variant{
					Condition:    cond,
					Constraint:   cid,
					Mask:         mask,
					Value:        buildRulesMapConstraintsArray(cv.([]gqt.Constraint)),
					Combinations: c,
				})
			default:
				switch cv := cv.(type) {
				case gqt.ValueArray:
					(*rm).rules[pathHash] = mergeVariants((*rm).rules[pathHash], Variant{
						Condition:    cond,
						Constraint:   cid,
						Mask:         mask,
						Value:        buildRulesMapConstraintsArray(cv.Items),
						Combinations: c,
					})
				case gqt.EnumValue:
					(*rm).rules[pathHash] = mergeVariants((*rm).rules[pathHash], Variant{
						Condition:    cond,
						Constraint:   cid,
						Mask:         mask,
						Value:        []byte(cv),
						Combinations: c,
					})
				default:
					(*rm).rules[pathHash] = mergeVariants((*rm).rules[pathHash], Variant{
						Condition:    cond,
						Constraint:   cid,
						Mask:         mask,
						Value:        cv,
						Combinations: c,
					})
				}
			}
		}
	}

	return leafs, nil
}

func buildRulesMapConstraintsElem(constraint gqt.Constraint) (el Elem) {
	var cv any
	var cid Constraint

	cid, cv = ConstraintIdAndValue(constraint)

	switch cid {
	case ConstraintMap:
		el = Elem{
			Constraint: cid,
			Value:      buildRulesMapConstraintsElem(cv),
		}
	case ConstraintOr, ConstraintAnd:
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
		case gqt.EnumValue:
			el = Elem{
				Constraint: cid,
				Value:      []byte(cv),
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
	obj = Object{}
	for _, constraint := range constraints {
		obj[constraint.Name] = buildRulesMapConstraintsElem(constraint.Value)
	}

	return
}

func mergeVariants(variants []Variant, x Variant) []Variant {
	merge := func(i int) {
		variants[i].Mask = variants[i].Mask.Or(x.Mask)
		variants[i].Combinations = append(variants[i].Combinations, x.Combinations...)
	}

	for i := 0; i < len(variants); i++ {
		if variants[i].Constraint == x.Constraint {
			switch vt := variants[i].Value.(type) {
			case Elem:
				switch xt := x.Value.(type) {
				case Elem:
					if vt.Equal(xt) {
						merge(i)
						return variants
					}
				}
			case Array:
				switch xt := x.Value.(type) {
				case Array:
					if vt.Equal(xt) {
						merge(i)
						return variants
					}
				}
			default:
				if xb, ok := x.Value.([]byte); ok {
					if bytes.Equal(variants[i].Value.([]byte), xb) {
						merge(i)
						return variants
					}
				} else if variants[i].Value == x.Value {
					merge(i)
					return variants
				}
			}
		}
	}
	variants = append(variants, x)

	return variants
}

// ConstraintIdAndValue returns constraint Id and Value.
func ConstraintIdAndValue(c gqt.Constraint) (Constraint, any) {
	switch c := c.(type) {
	case gqt.ConstraintOr:
		return ConstraintOr, c.Constraints
	case gqt.ConstraintAnd:
		return ConstraintAnd, c.Constraints
	case gqt.ConstraintMap:
		return ConstraintMap, c.Constraint
	case gqt.ConstraintAny:
		return ConstraintAny, nil
	case gqt.ConstraintValEqual:
		if s, ok := c.Value.(string); ok {
			return ConstraintValEqual, []byte(s)
		}
		return ConstraintValEqual, c.Value
	case gqt.ConstraintValNotEqual:
		if s, ok := c.Value.(string); ok {
			return ConstraintValNotEqual, []byte(s)
		}
		return ConstraintValNotEqual, c.Value
	case gqt.ConstraintValGreater:
		return ConstraintValGreater, c.Value
	case gqt.ConstraintValLess:
		return ConstraintValLess, c.Value
	case gqt.ConstraintValGreaterOrEqual:
		return ConstraintValGreaterOrEqual, c.Value
	case gqt.ConstraintValLessOrEqual:
		return ConstraintValLessOrEqual, c.Value
	case gqt.ConstraintBytelenEqual:
		return ConstraintBytelenEqual, c.Value
	case gqt.ConstraintBytelenNotEqual:
		return ConstraintBytelenNotEqual, c.Value
	case gqt.ConstraintBytelenGreater:
		return ConstraintBytelenGreater, c.Value
	case gqt.ConstraintBytelenLess:
		return ConstraintBytelenLess, c.Value
	case gqt.ConstraintBytelenGreaterOrEqual:
		return ConstraintBytelenGreaterOrEqual, c.Value
	case gqt.ConstraintBytelenLessOrEqual:
		return ConstraintBytelenLessOrEqual, c.Value
	case gqt.ConstraintLenEqual:
		return ConstraintLenEqual, c.Value
	case gqt.ConstraintLenNotEqual:
		return ConstraintLenNotEqual, c.Value
	case gqt.ConstraintLenGreater:
		return ConstraintLenGreater, c.Value
	case gqt.ConstraintLenLess:
		return ConstraintLenLess, c.Value
	case gqt.ConstraintLenGreaterOrEqual:
		return ConstraintLenGreaterOrEqual, c.Value
	case gqt.ConstraintLenLessOrEqual:
		return ConstraintLenLessOrEqual, c.Value
	default:
		return ConstraintUnknown, nil
	}
}

// Match returns the ID of the first matching template or "" if none was matched.
func (rm *RulesMap) Match(
	variableValues [][]gqlparse.Token,
	queryType gqlscan.Token,
	selectionSet []gqlparse.Token,
) (id string) {
	rm.FindMatch(variableValues, queryType, selectionSet, func(mask *bitmask.Set) {
		if mask.Size() > 0 {
			mask.Visit(func(n int) (skip bool) {
				id = rm.templateIDs[n]
				return true
			})
		}
	})
	return id
}

// MatchAll calls fn for every matching template.
func (rm *RulesMap) MatchAll(
	variableValues [][]gqlparse.Token,
	queryType gqlscan.Token,
	selectionSet []gqlparse.Token,
	fn func(id string),
) {
	rm.FindMatch(variableValues, queryType, selectionSet, func(mask *bitmask.Set) {
		mask.VisitAll(func(n int) {
			fn(rm.templateIDs[n])
		})
	})
}

// FindMatch matches query to the rules.
func (rm *RulesMap) FindMatch(
	variableValues [][]gqlparse.Token,
	queryType gqlscan.Token,
	selectionSet []gqlparse.Token,
	fn func(mask *bitmask.Set),
) {
	var qpCount int
	rm.matchCounter.Reset()
	rm.mask.Reset()
	rm.rejected.Reset()
	memset(rm.combinationCounters, 0)
	rm.qmake.ParseQuery(variableValues, queryType, selectionSet, func(qp pquery.QueryPart) (stop bool) {
		qpCount++
		if rn, ok := rm.rules[qp.Hash]; ok {
			if len(rn) > 0 {
				var match bool
				for _, v := range rn {
					if len(v.Combinations) > 0 {
						if qp.ArgLeafIdx < 1 {
							var depth int
							for _, c := range v.Combinations {
								if rm.combinationCounters[c.Index] == 0 {
									depth = c.Depth
								}
								for i := c.Index - depth; i <= c.Index; i++ {
									rm.combinationCounters[i]++
									if rm.combinations[i] < rm.combinationCounters[i] {
										rm.rejected.Add(c.RuleIndex)
									}
								}
							}
						}
					}

					if v.Compare(qp.Value) {
						match = true
						rm.mask.SetOr(rm.mask, v.Mask)
						v.Mask.Visit(func(x int) (skip bool) {
							rm.matchCounter.SetFn(x, 1, func(value *int) { *value++ })
							return false
						})
					}
				}
				if !match {
					rm.mask.Reset()
					return true
				}
			}
		} else {
			rm.mask.Reset()
			return true
		}

		return false
	})
	for _, el := range rm.matchCounter.A {
		if el.Value < qpCount {
			rm.rejected.Add(el.Key)
		}
	}
	rm.mask.SetAndNot(rm.mask, rm.rejected)

	if rm.mask.Empty() {
		rm.mask.Reset()
	}

	fn(rm.mask)
}

// CompareValues compares two values according to the provided constraint.
func CompareValues(constraint Constraint, a any, b any) bool {
	switch constraint {
	case None, ConstraintAny:
		return true
	case ConstraintValEqual:
		if b, ok := b.([]byte); ok {
			return bytes.Equal(b, a.([]byte))
		}
		return b == a
	case ConstraintValNotEqual:
		if b, ok := b.([]byte); ok {
			return !bytes.Equal(b, a.([]byte))
		}
		return b != a
	case ConstraintValGreater, ConstraintValLess,
		ConstraintValGreaterOrEqual, ConstraintValLessOrEqual:
		switch vala := a.(type) {
		case int64:
			valb, ok := b.(int64)
			if !ok {
				return false
			}
			switch constraint {
			case ConstraintValGreater:
				return valb > vala
			case ConstraintValLess:
				return valb < vala
			case ConstraintValGreaterOrEqual:
				return valb >= vala
			case ConstraintValLessOrEqual:
				return valb <= vala
			}
		case float64:
			valb, ok := b.(float64)
			if !ok {
				return false
			}
			switch constraint {
			case ConstraintValGreater:
				return valb > vala
			case ConstraintValLess:
				return valb < vala
			case ConstraintValGreaterOrEqual:
				return valb >= vala
			case ConstraintValLessOrEqual:
				return valb <= vala
			}
		}
	case ConstraintBytelenEqual, ConstraintBytelenNotEqual,
		ConstraintBytelenGreater, ConstraintBytelenLess,
		ConstraintBytelenGreaterOrEqual, ConstraintBytelenLessOrEqual:
		vala, ok := a.(uint)
		if !ok {
			return false
		}
		valb, ok := b.([]byte)
		if !ok {
			return false
		}
		switch constraint {
		case ConstraintBytelenEqual:
			return len(valb) == int(vala)
		case ConstraintBytelenNotEqual:
			return len(valb) != int(vala)
		case ConstraintBytelenGreater:
			return len(valb) > int(vala)
		case ConstraintBytelenLess:
			return len(valb) < int(vala)
		case ConstraintBytelenGreaterOrEqual:
			return len(valb) >= int(vala)
		case ConstraintBytelenLessOrEqual:
			return len(valb) <= int(vala)
		}
	case ConstraintLenEqual, ConstraintLenNotEqual,
		ConstraintLenGreater, ConstraintLenLess,
		ConstraintLenGreaterOrEqual, ConstraintLenLessOrEqual:
		ca, ok := a.(uint)
		if !ok {
			return false
		}
		bi, ok := b.(*[]any)
		if !ok {
			return false
		}
		switch constraint {
		case ConstraintLenEqual:
			return len(*bi) == int(ca)
		case ConstraintLenNotEqual:
			return len(*bi) != int(ca)
		case ConstraintLenGreater:
			return len(*bi) > int(ca)
		case ConstraintLenLess:
			return len(*bi) < int(ca)
		case ConstraintLenGreaterOrEqual:
			return len(*bi) >= int(ca)
		case ConstraintLenLessOrEqual:
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
	case ConstraintMap:
		switch xt := x.(type) {
		case *[]any:
			for _, el := range *xt {
				switch vt := v.Value.(type) {
				case Elem:
					if !vt.Compare(el) {
						return false
					}
				}
			}
			return true
		}
	case ConstraintOr:
		for _, el := range v.Value.(Array) {
			if el.Compare(x) {
				return true
			}
		}
		return false
	case ConstraintAnd:
		for _, el := range v.Value.(Array) {
			if !el.Compare(x) {
				return false
			}
		}
		return true
	default:
		neq := v.Constraint == ConstraintValNotEqual
		switch vt := v.Value.(type) {
		case Array:
			switch xt := x.(type) {
			case *[]any:
				return vt.Compare(*xt) != neq
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
	case ConstraintMap:
		switch xt := x.(type) {
		case *[]any:
			for _, el := range *xt {
				switch et := e.Value.(type) {
				case Elem:
					return et.Compare(el)
				}
			}
		}
	case ConstraintOr:
		for _, el := range e.Value.(Array) {
			if el.Compare(x) {
				return true
			}
		}
		return false
	case ConstraintAnd:
		for _, el := range e.Value.(Array) {
			if !el.Compare(x) {
				return false
			}
		}
		return true
	default:
		neq := e.Constraint == ConstraintValNotEqual
		switch et := e.Value.(type) {
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
		_, _ = w.Write([]byte(":"))
		if len(rn) > 0 {
			_, _ = w.Write([]byte("\n"))
			for _, v := range rn {
				v.print(w, indent+4)
			}
		}
	}
}

func (v *Variant) print(w io.Writer, indent uint) {
	PrintNSpaces(w, indent)
	_, _ = w.Write(append([]byte(ConstraintLookup[v.Constraint]), []byte(": ")...))
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
	_, _ = w.Write(append([]byte(ConstraintLookup[e.Constraint]), []byte(":\n")...))
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
		_, _ = w.Write(append([]byte(ConstraintLookup[el.Constraint]), []byte(":\n")...))
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
		_, _ = w.Write(append([]byte(ConstraintLookup[el.Constraint]), []byte(":\n")...))
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

func memset[T comparable](a []T, v T) {
	if len(a) == 0 {
		return
	}
	a[0] = v
	for i := 1; i < len(a); i *= 2 {
		copy(a[i:], a[:i])
	}
}

// Constraint is a constraint simplified abstraction.
type Constraint uint16

const (
	None Constraint = iota
	ConstraintUnknown
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
	None:                            "NoConstraint",
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
