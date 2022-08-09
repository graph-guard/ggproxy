package hmap

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"

	"github.com/graph-guard/gguard-proxy/utilities/unsafe"
	"github.com/graph-guard/gguard-proxy/utilities/xxhash"

	"github.com/graph-guard/gqlscan"
	"github.com/graph-guard/gqt"
)

var (
	ErrSyntax            = errors.New("syntax error")
	ErrInputExceedsLimit = errors.New("input exceeds limit")
)

type candidate struct {
	Template      gqt.Doc
	TemplateIndex int
	Constraints   []Constraint
}

type Matcher struct {
	hashSeed uint64

	// templateIndexCounter is the index of the last
	// template (both query)
	templateIndexCounter int

	// queryTemplates maps hashes to query candidates
	queryTemplates map[uint64][]*candidate

	// mutationTemplates maps hashes to mutation candidates
	mutationTemplates map[uint64][]*candidate

	tBuffer []token
	iBuffer []irange
	hBuffer []uint64
	stack   []xxhash.Hash
}

// New initializes a new matcher instance.
// maxInputLen will be at least 128.
func New(maxInputLen int) *Matcher {
	if maxInputLen < 128 {
		maxInputLen = 128
	}
	return &Matcher{
		templateIndexCounter: -1,
		// TODO: determine best seed based on collisions (if any)
		hashSeed:          0,
		queryTemplates:    make(map[uint64][]*candidate),
		mutationTemplates: make(map[uint64][]*candidate),
		tBuffer:           make([]token, maxInputLen),
		iBuffer:           make([]irange, maxInputLen),
		hBuffer:           make([]uint64, maxInputLen),
		stack:             make([]xxhash.Hash, 0, 1024),
	}
}

func (m *Matcher) GetTemplateByIndex(
	index int,
) (hash uint64, template gqt.Doc) {
	if index < 0 || index > m.templateIndexCounter {
		return 0, nil
	}
	for _, mp := range [2]map[uint64][]*candidate{
		m.queryTemplates, m.mutationTemplates,
	} {
		for hash, candidates := range mp {
			for _, candidate := range candidates {
				if candidate.TemplateIndex == index {
					return hash, candidate.Template
				}
			}
		}
	}
	return 0, nil
}

func (m *Matcher) RemoveTemplate(index int) bool {
	if index < 0 || index > m.templateIndexCounter {
		return false
	}
	for _, mp := range [2]map[uint64][]*candidate{
		m.queryTemplates, m.mutationTemplates,
	} {
		for hash, candidates := range mp {
			for ci, candidate := range candidates {
				if candidate.TemplateIndex == index {
					if len(candidates) < 2 {
						delete(mp, hash)
						return true
					}
					mp[hash] = append(candidates[:ci], candidates[ci+1:]...)
					return true
				}
			}
		}
	}
	return false
}

func (m *Matcher) AddTemplate(d gqt.Doc) {
	var hashmap map[uint64][]*candidate
	switch d := d.(type) {
	case gqt.DocQuery:
		hashmap = m.queryTemplates
	case gqt.DocMutation:
		hashmap = m.mutationTemplates
	default:
		panic(fmt.Errorf("unsupported GQT document type: %q", d))
	}
	constraints := extractInputConstraints(d)
	hash := m.computeTemplateHash(d)
	m.templateIndexCounter++
	hashmap[hash] = append(hashmap[hash], &candidate{
		TemplateIndex: m.templateIndexCounter,
		Constraints:   constraints,
	})
}

func (m *Matcher) Match(
	query, varsJSON []byte,
) (templateIndex int, err error) {
	if len(query) > cap(m.tBuffer) {
		return -1, ErrInputExceedsLimit
	}

	// Reset stack
	m.stack = m.stack[:0]

	tIndex, iIndex, hIndex, inVal := -1, 0, 0, false
	if err := gqlscan.ScanAll(query, func(i *gqlscan.Iterator) {
		tIndex++
		t := i.Token()
		m.tBuffer[tIndex] = token{
			Type:  t,
			Value: i.Value(),
		}
		switch t {
		case gqlscan.TokenArgName:
			if inVal {
				m.iBuffer[iIndex-1].End = tIndex
			}
			m.iBuffer[iIndex] = irange{Start: tIndex + 1}
			iIndex++
			inVal = true
		case gqlscan.TokenArgListEnd:
			m.iBuffer[iIndex-1].End = tIndex
			inVal = false
		}
	}); err.IsErr() {
		return -1, ErrSyntax
	}

	{ // Compute hashes
		tokens := m.tBuffer[:tIndex+1]
		// Push stack
		m.stack = append(m.stack, xxhash.New(m.hashSeed))
		// Skip until first field
		tokens = tokens[2:]
		for i := range tokens {
			if tokens[i].Type == gqlscan.TokenField {
				tokens = tokens[i:]
				break
			}
		}
		for i := range tokens {
			switch tokens[i].Type {
			case gqlscan.TokenField:
				// From stack top
				h := m.stack[len(m.stack)-1]
				xxhash.Write(&h, delimiterField)
				xxhash.Write(&h, tokens[i].Value)
				// Push stack
				m.stack = append(m.stack, h)
				if n := tokens[i+1]; n.Type != gqlscan.TokenArgList &&
					n.Type != gqlscan.TokenSet {
					// Pop stack
					if len(m.stack) > 0 {
						m.stack = m.stack[:len(m.stack)-1]
					}
				}
				m.hBuffer[hIndex] = h.Sum64()
				hIndex++
			case gqlscan.TokenArgListEnd:
				if tokens[i+1].Type != gqlscan.TokenSet {
					// Pop stack
					if len(m.stack) > 0 {
						m.stack = m.stack[:len(m.stack)-1]
					}
				}
			case gqlscan.TokenSetEnd:
				// Pop stack
				if len(m.stack) > 0 {
					m.stack = m.stack[:len(m.stack)-1]
				}
			case gqlscan.TokenArgName:
				// From stack top
				h := m.stack[len(m.stack)-1]
				xxhash.Write(&h, delimiterArg)
				xxhash.Write(&h, tokens[i].Value)
				m.hBuffer[hIndex] = h.Sum64()
				hIndex++
			}
		}
	}

	var controlHash uint64
	{ // Compute control hash
		h := xxhash.New(m.hashSeed)
		buf := [8]byte{}
		for _, hash := range m.hBuffer[:hIndex] {
			binary.LittleEndian.PutUint64(buf[:], hash)
			xxhash.Write8(&h, buf)
		}
		controlHash = h.Sum64()
	}

	// TODO: add support for named queries and operation name
	var hashmap map[uint64][]*candidate
	switch m.tBuffer[0].Type {
	case gqlscan.TokenDefQry:
		// Query
		hashmap = m.queryTemplates
	case gqlscan.TokenDefMut:
		// Mutation
		hashmap = m.mutationTemplates
	default:
		return -1, fmt.Errorf(
			"unsupported query type %q", m.tBuffer[0].Type.String(),
		)
	}

	candidates, ok := hashmap[controlHash]
	if !ok {
		// No match
		return -1, nil
	}

CANDIDATES:
	for _, candidate := range candidates {
		for i, c := range candidate.Constraints {
			if i >= len(m.iBuffer) {
				// Missing inputs
				continue CANDIDATES
			}
			r := m.iBuffer[i]
			if !MatchValue(m.tBuffer[r.Start:r.End], c) {
				// Inputs don't match constraint
				continue CANDIDATES
			}
		}
		return candidate.TemplateIndex, nil
	}
	return -1, nil
}

func extractInputConstraints(d gqt.Doc) (c []Constraint) {
	var traverse func(node any)
	traverse = func(node any) {
		switch n := node.(type) {
		case gqt.DocQuery:
			for _, s := range n.Selections {
				traverse(s)
			}
		case gqt.DocMutation:
			for _, s := range n.Selections {
				traverse(s)
			}
		case gqt.SelectionField:
			for _, ic := range n.InputConstraints {
				traverse(ic)
			}
			for _, s := range n.Selections {
				traverse(s)
			}
		case gqt.InputConstraint:
			t := translateConstraint(n.Constraint)
			c = append(c, t)
		default:
			panic(fmt.Errorf("unsupported GQT node type: %T", node))
		}
	}
	traverse(d)
	return c
}

func translateConstraint(c gqt.Constraint) Constraint {
	switch c := c.(type) {
	case gqt.ConstraintAnd:
		// TODO
		panic("not yet supported")
	case gqt.ConstraintOr:
		// TODO
		panic("not yet supported")

	case gqt.ConstraintBytelenEqual:
		// TODO
		panic("not yet supported")
	case gqt.ConstraintBytelenNotEqual:
		// TODO
		panic("not yet supported")
	case gqt.ConstraintBytelenGreater:
		// TODO
		panic("not yet supported")
	case gqt.ConstraintBytelenGreaterOrEqual:
		// TODO
		panic("not yet supported")
	case gqt.ConstraintBytelenLess:
		// TODO
		panic("not yet supported")
	case gqt.ConstraintBytelenLessOrEqual:
		// TODO
		panic("not yet supported")

	case gqt.ConstraintLenEqual:
		// TODO
		panic("not yet supported")
	case gqt.ConstraintLenNotEqual:
		// TODO
		panic("not yet supported")
	case gqt.ConstraintLenGreater:
		// TODO
		panic("not yet supported")
	case gqt.ConstraintLenGreaterOrEqual:
		// TODO
		panic("not yet supported")
	case gqt.ConstraintLenLess:
		// TODO
		panic("not yet supported")
	case gqt.ConstraintLenLessOrEqual:
		// TODO
		panic("not yet supported")

	case gqt.ConstraintValGreater:
		return translateValueNumeric[Gr](c.Value)
	case gqt.ConstraintValGreaterOrEqual:
		return translateValueNumeric[GrOrEq](c.Value)
	case gqt.ConstraintValLess:
		return translateValueNumeric[Le](c.Value)
	case gqt.ConstraintValLessOrEqual:
		return translateValueNumeric[LeOrEq](c.Value)

	case gqt.ConstraintAny:
		// TODO
		panic("not yet supported")

	case gqt.ConstraintValEqual:
		return translateValueAnyType[Eq](c.Value)

	case gqt.ConstraintValNotEqual:
		return translateValueAnyType[NotEq](c.Value)

	default:
		panic(fmt.Errorf("unsupported GQT type: %T", c))
	}
}

func (m *Matcher) computeTemplateHash(d gqt.Doc) uint64 {
	hash := xxhash.New(m.hashSeed)
	var traverse func(parent string, node any)
	traverse = func(parent string, node any) {
		switch n := node.(type) {
		case gqt.DocQuery:
			for _, s := range n.Selections {
				traverse("", s)
			}
		case gqt.DocMutation:
			for _, s := range n.Selections {
				traverse("", s)
			}
		case gqt.SelectionField:
			parent += string(delimiterField) + n.Name
			h := xxhash.New(m.hashSeed)
			xxhash.Write(&h, parent)
			writeHashToHash(&hash, h)
			for _, c := range n.InputConstraints {
				traverse(parent, c)
			}
			for _, s := range n.Selections {
				traverse(parent, s)
			}
		case gqt.InputConstraint:
			parent += string(delimiterArg) + n.Name
			h := xxhash.New(m.hashSeed)
			xxhash.Write(&h, parent)
			writeHashToHash(&hash, h)
		default:
			panic(fmt.Errorf("unsupported GQT node type: %T", node))
		}
	}
	traverse("", d)
	return hash.Sum64()
}

func writeHashToHash(to *xxhash.Hash, hash xxhash.Hash) {
	buf := [8]byte{}
	binary.LittleEndian.PutUint64(buf[:], hash.Sum64())
	xxhash.Write8(to, buf)
}

func translateValueAnyType[T ~struct{ Value any }](v gqt.Value) Constraint {
	var x any
	switch v := v.(type) {
	case nil:
	case bool:
		x = v
	case string:
		x = v
	case float64:
		str := []byte(strconv.FormatFloat(v, 'f', -1, 64))
		x = Number{Val: str}
	case gqt.ValueArray:
		a := make(Array, len(v.Items))
		for i, c := range v.Items {
			a[i] = translateConstraint(c)
		}
		x = a
	case gqt.ValueObject:
		o := Object{Fields: make([]ObjectField, len(v.Fields))}
		for i, c := range v.Fields {
			o.Fields[i] = ObjectField{
				Name:  c.Name,
				Value: translateConstraint(c.Value),
			}
		}
		x = o
	default:
		panic(fmt.Errorf("unsupported value type: %T", v))
	}

	return Constraint{T{Value: x}}
}

func translateValueNumeric[T ~struct{ Value float64 }](
	v gqt.Value,
) Constraint {
	switch v := v.(type) {
	case float64:
		return Constraint{T{Value: v}}
	default:
		panic(fmt.Errorf("unsupported value type: %T", v))
	}
}

// func readUntil(tokens []token, t gqlscan.Token) (until, after []token) {
// 	for i := range tokens {
// 		if tokens[i].Type == t {
// 			return tokens[:i+1], tokens[i+1:]
// 		}
// 	}
// 	return tokens, nil
// }

// Constraint can be either of:
//
//	Eq
//	NotEq
//	Gr
//	GrOrEq
//	Le
//	LeOrEq
type Constraint struct{ any }

func (c Constraint) String() string {
	switch c := c.any.(type) {
	case Eq:
		return "is " + c.String()
	case NotEq:
		return "is " + c.String()
	case Gr:
		return "is " + c.String()
	case GrOrEq:
		return "is " + c.String()
	case Le:
		return "is " + c.String()
	case LeOrEq:
		return "is " + c.String()
	}
	panic(fmt.Errorf("unsupported constraint type: %T", c))
}

// Eq expects equality against Value which can be either of:
//
//	nil (null value)
//	bool
//	string
//	Number
//	Array
//	ArrayMap
//	Object
type Eq struct{ Value any }

func (e Eq) String() string {
	switch v := e.Value.(type) {
	case nil:
		return "== null"
	case bool:
		return fmt.Sprintf("== %t", v)
	case string:
		return fmt.Sprintf("== %q", v)
	case Number:
		return fmt.Sprintf("== %s", v.Val)
	case Array:
		return fmt.Sprintf("== %s", v.String())
	case ArrayMap:
		return fmt.Sprintf("== %s", v.String())
	case Object:
		return fmt.Sprintf("== %s", v.String())
	}
	panic(fmt.Errorf("unsupported value type: %T", e.Value))
}

// Gr expects expects the input value to be greater than Value.
type Gr struct{ Value float64 }

func (c Gr) String() string {
	return fmt.Sprintf("> %f", c.Value)
}

// GrOrEq expects expects the input value to be greater than or equal Value.
type GrOrEq struct{ Value float64 }

func (c GrOrEq) String() string {
	return fmt.Sprintf(">= %f", c.Value)
}

// Le expects expects the input value to be less than Value.
type Le struct{ Value float64 }

func (c Le) String() string {
	return fmt.Sprintf("< %f", c.Value)
}

// LeOrEq expects expects the input value to be less than or equal Value.
type LeOrEq struct{ Value float64 }

func (c LeOrEq) String() string {
	return fmt.Sprintf("<= %f", c.Value)
}

// NotEq is same as Eq but negated
type NotEq struct{ Value any }

func (n NotEq) String() string {
	switch v := n.Value.(type) {
	case nil:
		return "!= null"
	case bool:
		return fmt.Sprintf("!= %t", v)
	case string:
		return fmt.Sprintf("!= %q", v)
	case Number:
		return fmt.Sprintf("!= %s", v.Val)
	case Array:
		return fmt.Sprintf("!= %s", v.String())
	case ArrayMap:
		return fmt.Sprintf("!= %s", v.String())
	case Object:
		return fmt.Sprintf("!= %s", v.String())
	}
	panic(fmt.Errorf("unsupported value type: %T", n.Value))
}

type Number struct{ Val []byte }

type Array []Constraint

func (a Array) String() string {
	s := "["
	for i, c := range a {
		s += fmt.Sprintf("%d: ", i)
		s += c.String()
		if i+1 < len(a) {
			s += ", "
		}
	}
	s += "]"
	return s
}

// ArrayMap is a constraint that applies to all items in an array
type ArrayMap struct{ Items Constraint }

func (a ArrayMap) String() string {
	return "[... " + a.Items.String() + "]"
}

type Object struct{ Fields []ObjectField }

func (o Object) String() string {
	s := "{"
	for i, f := range o.Fields {
		s += f.String()
		if i+1 < len(o.Fields) {
			s += ", "
		}
	}
	s += "}"
	return s
}

type ObjectField struct {
	Name  string
	Value Constraint
}

func (f ObjectField) String() string {
	return f.Name + " " + f.Value.String()
}

type token struct {
	Type  gqlscan.Token
	Value []byte
}

func (t token) String() string {
	if t.Type == 0 {
		return "{}"
	}
	return fmt.Sprintf("{%s: %q}", t.Type.String(), string(t.Value))
}

type irange struct {
	Start, End int
}

func MatchValue(tokens []token, c Constraint) bool {
	_, result := matchValue(tokens, c)
	return result
}

func matchValue(tokens []token, c Constraint) (after []token, result bool) {
	defer func() {
		if _, ok := c.any.(NotEq); ok {
			result = !result
		}
	}()

	var val any
	switch c := c.any.(type) {
	case Eq:
		val = c.Value

	case NotEq:
		val = c.Value

	case Gr:
		t := tokens[0]
		if t.Type != gqlscan.TokenInt && t.Type != gqlscan.TokenFloat {
			// Unexpected value type
			return nil, false
		}
		n, err := strconv.ParseFloat(unsafe.B2S(t.Value), 64)
		if err != nil {
			return nil, false
		}
		return tokens[1:], n > c.Value

	case GrOrEq:
		t := tokens[0]
		if t.Type != gqlscan.TokenInt && t.Type != gqlscan.TokenFloat {
			// Unexpected value type
			return nil, false
		}
		n, err := strconv.ParseFloat(unsafe.B2S(t.Value), 64)
		if err != nil {
			return nil, false
		}
		return tokens[1:], n >= c.Value

	case Le:
		t := tokens[0]
		if t.Type != gqlscan.TokenInt && t.Type != gqlscan.TokenFloat {
			// Unexpected value type
			return nil, false
		}
		n, err := strconv.ParseFloat(unsafe.B2S(t.Value), 64)
		if err != nil {
			return nil, false
		}
		return tokens[1:], n < c.Value

	case LeOrEq:
		t := tokens[0]
		if t.Type != gqlscan.TokenInt && t.Type != gqlscan.TokenFloat {
			// Unexpected value type
			return nil, false
		}
		n, err := strconv.ParseFloat(unsafe.B2S(t.Value), 64)
		if err != nil {
			return nil, false
		}
		return tokens[1:], n <= c.Value

	default:
		panic(fmt.Errorf("unsupported constraint type: %T", c))
	}

	switch v := val.(type) {
	case nil:
		if tokens[0].Type != gqlscan.TokenNull {
			// Unexpected value type
			return nil, false
		}
		return tokens[1:], true

	case bool:
		switch tokens[0].Type {
		case gqlscan.TokenFalse:
			return tokens[1:], v == false
		case gqlscan.TokenTrue:
			return tokens[1:], v == true
		default:
			// Unexpected value type
			return nil, false
		}

	case Number:
		if tokens[0].Type != gqlscan.TokenInt &&
			tokens[0].Type != gqlscan.TokenFloat {
			// Unexpected value type
			return nil, false
		}
		return tokens[1:], bytes.Equal(v.Val, tokens[0].Value)

	case string:
		if tokens[0].Type != gqlscan.TokenStr {
			// Unexpected value type
			return nil, false
		}
		return tokens[1:], v == string(tokens[0].Value)

	case ArrayMap:
		if tokens[0].Type != gqlscan.TokenArr {
			// Unexpected value type
			return nil, false
		}
		tokens = tokens[1:]
		for len(tokens) > 0 {
			if tokens[0].Type == gqlscan.TokenArrEnd {
				return tokens[1:], true
			}
			if tokens, result = matchValue(tokens, v.Items); !result {
				return nil, false
			}
		}
		return tokens, true

	case Array:
		if tokens[0].Type != gqlscan.TokenArr {
			// Unexpected value type
			return nil, false
		}
		tokens = tokens[1:]
		for i := 0; len(tokens) > 0; i++ {
			if tokens[0].Type == gqlscan.TokenArrEnd {
				if i+1 < len(v) {
					// Missing items
					return nil, false
				}
				return tokens[1:], true
			}
			if i >= len(v) {
				// More items than required
				return nil, false
			}
			if tokens, result = matchValue(tokens, v[i]); !result {
				return nil, false
			}
		}
		return tokens, true

	case Object:
		if tokens[0].Type != gqlscan.TokenObj {
			// Unexpected value type
			return nil, false
		}
		tokens = tokens[1:]
		for i := 0; len(tokens) > 0; i++ {
			if tokens[0].Type == gqlscan.TokenObjEnd {
				if i+1 < len(v.Fields) {
					// Input is missing required fields
					return nil, false
				}
				tokens = tokens[1:]
				return tokens, true
			}
			f := v.Fields[i]
			if string(tokens[0].Value) != f.Name {
				// Unexpected field
				return nil, false
			}
			tokens = tokens[1:]
			tokens, result = matchValue(tokens, f.Value)
			if !result {
				return nil, false
			}
		}
	}

	return nil, false
}

var (
	delimiterField = []byte(".")
	delimiterArg   = []byte("|")
)
