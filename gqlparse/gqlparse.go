// Package gqlparse provides a GraphQL parser that validates queries,
// ignores irrelevant operations inlines fragments and
// references variable values.
package gqlparse

import (
	"fmt"
	"io"
	"strings"

	"github.com/graph-guard/ggproxy/gqlparse/internal/graph"
	"github.com/graph-guard/ggproxy/utilities/container/hamap"
	"github.com/graph-guard/ggproxy/utilities/segmented"
	"github.com/graph-guard/ggproxy/utilities/stack"
	"github.com/graph-guard/ggproxy/utilities/unsafe"
	"github.com/graph-guard/gqlscan"
	"github.com/tidwall/gjson"
)

type Token struct {
	// IDs below 100 are reserved gqlscan type identifiers.
	// IDs above 99 are variable value indexes.
	ID    gqlscan.Token
	Value []byte
}

func (t Token) String() string {
	if t.Value == nil {
		return t.ID.String()
	}
	return fmt.Sprintf("%s (%q)", t.ID.String(), string(t.Value))
}

// VariableIndex returns the index of the varible value if
// its a variable index token, otherwise returns -1.
func (t Token) VariableIndex() (index int) {
	if t.ID >= TokenTypeValIndexOffset {
		return int(t.ID - TokenTypeValIndexOffset)
	}
	return -1
}

// MakeVariableIndexToken creates a variable index token.
func MakeVariableIndexToken(index int, name string) Token {
	return Token{
		ID:    gqlscan.Token(TokenTypeValIndexOffset + index),
		Value: []byte(name),
	}
}

// TokenTypeValIndexOffset defines the offset that needs to be subtracted
// from Token.ID in order to get the index of the value when Token.ID is > 99
const TokenTypeValIndexOffset = 100

type indexRange struct {
	IndexStart int
	IndexEnd   int
}

type fragDef struct {
	indexRange
	Used bool
}

type varDecl struct {
	Name         []byte
	Type         indexRange
	DefaultValue indexRange
}

// typeUnion is a union that defines either of (depending on combination):
//   - nullable type (Nullable: true; Array: false; TypeName: !nil)
//   - not nullable type (Nullable: false; Array: false; TypeName: !nil)
//   - nullable array (Nullable: true; Array: true; TypeName: nil)
//   - not nullable array (Nullable: false; Array: true; TypeName: nil)
type typeUnion struct {
	Nullable bool
	Array    bool
	TypeName []byte
}

// NewParser creates a new parser instance.
// It's adviced to create only one parser per goroutine
// as calling (*Parser).Parse will reset it.
func NewParser() *Parser {
	return &Parser{
		gi:        graph.NewInspector(),
		buffer:    make([]Token, 0),
		bufferOpr: make([]Token, 0),
		buffer2:   make([]Token, 0),
		fragDefs: hamap.New[[]byte, fragDef](
			graph.MaxFragments, nil,
		),
		fragsConstructed:   segmented.New[[]byte, Token](),
		entryFrags:         hamap.New[[]byte, struct{}](0, nil),
		varOrder:           make([][]byte, 0),
		varsDecls:          hamap.New[[]byte, varDecl](0, nil),
		varValues:          make([][]Token, 0),
		varsConstructed:    segmented.New[[]byte, Token](),
		typeStack:          stack.New[typeUnion](0),
		operations:         make([]indexRange, 0),
		fragmentGraphEdges: make([]graph.Edge, 0),
		errFragLimitExceeded: ErrorFragLimitExceeded{
			Limit: graph.MaxFragments,
		},
	}
}

type Parser struct {
	// buffer holds the original source tokens
	buffer []Token

	// bufferOpr is used during operation selection and inlining
	bufferOpr []Token

	// buffer2 is used for JSON value tokenization
	buffer2 []Token

	// gi is used during fragment construction and fragment cycle detection
	gi *graph.Inspector

	// ordered is used for fragment inlining, it stores the
	// fragment names in the order of dependency.
	ordered [][]byte

	// fragDefs indexes all fragment definitions
	fragDefs *hamap.Map[[]byte, fragDef]

	// entryFrags indexes all entry level fragments.
	entryFrags *hamap.Map[[]byte, struct{}]

	// fragsConstructed stores all constructed structs
	fragsConstructed *segmented.Array[[]byte, Token]

	// varOrder defines the order of variable declarations
	varOrder [][]byte

	// varsDecls indexes all variable declarations
	varsDecls *hamap.Map[[]byte, varDecl]

	// varValues stores all constructed variable values
	varValues [][]Token

	// varsConstructed stores all constructed variable values
	varsConstructed *segmented.Array[[]byte, Token]

	// typeStack is used during variable default value parsing
	typeStack stack.Stack[typeUnion]

	// operations holds index ranges of all operation definitions
	operations []indexRange

	// fragmentGraphEdges buffers the edges that are passed to gi
	fragmentGraphEdges []graph.Edge

	errSyntax            ErrorSyntax
	errOprAnonNonExcl    ErrorOprAnonNonExcl
	errOprNotFound       ErrorOprNotFound
	errRedecOpr          ErrorRedecOpr
	errRedecFrag         ErrorRedecFrag
	errFragUnused        ErrorFragUnused
	errFragUndefined     ErrorFragUndefined
	errFragRecurse       ErrorFragRecurse
	errFragLimitExceeded ErrorFragLimitExceeded
	errRedeclVar         ErrorRedeclVar
	errUnexpValType      ErrorUnexpValType
	errVarUndeclared     ErrorVarUndeclared
	errVarUndefined      ErrorVarUndefined
	errVarJSONSyntax     ErrorVarJSONSyntax
	errVarJSONNotObj     ErrorVarJSONNotObj
}

func (r *Parser) reset() {
	r.buffer = r.buffer[:0]
	r.bufferOpr = r.bufferOpr[:0]
	r.ordered = r.ordered[:0]
	r.fragDefs.Reset()
	r.entryFrags.Reset()
	r.varOrder = r.varOrder[:0]
	r.varValues = r.varValues[:0]
	r.varsDecls.Reset()
	r.varsConstructed.Reset()
	r.fragsConstructed.Reset()
	r.operations = r.operations[:0]
	r.fragmentGraphEdges = r.fragmentGraphEdges[:0]
}

// Parse calls onSuccess in case of success where operation
// only contains the relevant set of tokens.
// onError is called in case of an error.
//
// WARNING: Data (including errors) provided to callbacks must not be
// aliased and used after Parse returns!
func (r *Parser) Parse(
	src, operationName, varsJSON []byte,
	onSuccess func(
		varValues [][]Token,
		operation []Token,
		selectionSet []Token,
	),
	onError func(err error),
) {
	r.reset()
	var isErr bool
	var recentDef gqlscan.Token
	var stackCounter int
	var fragStackCounter int
	var recentFragDef []byte

	if serr := gqlscan.Scan(src, func(i *gqlscan.Iterator) bool {
		r.buffer = append(r.buffer, Token{
			ID:    i.Token(),
			Value: i.Value(),
		})
		switch i.Token() {
		case gqlscan.TokenDefMut:
			recentDef = gqlscan.TokenDefMut
			r.operations = append(r.operations, indexRange{
				IndexStart: len(r.buffer) - 1,
			})

		case gqlscan.TokenDefQry:
			recentDef = gqlscan.TokenDefQry
			r.operations = append(r.operations, indexRange{
				IndexStart: len(r.buffer) - 1,
			})

		case gqlscan.TokenDefSub:
			recentDef = gqlscan.TokenDefSub
			r.operations = append(r.operations, indexRange{
				IndexStart: len(r.buffer) - 1,
			})

		case gqlscan.TokenFragInline:
			if i.Value() != nil {
				break
			}
			r.buffer = r.buffer[:len(r.buffer)-1]
			fragStackCounter++

		case gqlscan.TokenSet:
			if fragStackCounter > 0 {
				r.buffer = r.buffer[:len(r.buffer)-1]
				break
			}
			stackCounter++

		case gqlscan.TokenSetEnd:
			if fragStackCounter > 0 {
				r.buffer = r.buffer[:len(r.buffer)-1]
				fragStackCounter--
				break
			}
			stackCounter--
			if stackCounter < 1 {
				if recentDef == gqlscan.TokenDefMut ||
					recentDef == gqlscan.TokenDefSub ||
					recentDef == gqlscan.TokenDefQry {
					// End of mutation/subscription or query operation
					r.operations[len(r.operations)-1].IndexEnd = len(r.buffer)
				} else {
					// End of fragment definition
					r.fragDefs.GetFn(recentFragDef, func(v *fragDef) {
						v.IndexEnd = len(r.buffer)
					})
				}
				recentDef, recentFragDef = 0, nil
			}
		case gqlscan.TokenNamedSpread:
			if recentFragDef == nil {
				r.entryFrags.Set(
					r.buffer[len(r.buffer)-1].Value, struct{}{},
				)
			} else {
				r.fragmentGraphEdges = append(r.fragmentGraphEdges, graph.Edge{
					From: recentFragDef,
					To:   r.buffer[len(r.buffer)-1].Value,
				})
			}

		case gqlscan.TokenFragName:
			// Fragment definition
			if _, ok := r.fragDefs.Get(i.Value()); ok {
				// Found redeclared fragment
				isErr = true
				r.errRedecFrag.FragmentName = i.Value()
				onError(&r.errRedecFrag)
				return true
			}
			recentFragDef = i.Value()
			if r.fragDefs.Len() >= graph.MaxFragments {
				onError(&r.errFragLimitExceeded)
				isErr = true
				return true
			}
			r.fragDefs.Set(i.Value(), fragDef{
				indexRange: indexRange{
					IndexStart: len(r.buffer) - 1,
				},
			})
		}
		return false
	}); serr.IsErr() {
		if isErr {
			return
		}
		r.errSyntax.ScanErr = serr
		onError(&r.errSyntax)
		return
	}

	// Check for operation redeclaration and find selected operation
	oprIndex := -1
	for i := 0; i < len(r.operations); i++ {
		x := r.buffer[r.operations[i].IndexStart+1]
		if x.ID != gqlscan.TokenOprName {
			// Make sure this anonymous operation is
			// the only operation in the request
			if len(r.operations) > 1 {
				onError(&r.errOprAnonNonExcl)
				return
			}
			if len(operationName) > 0 {
				r.errOprNotFound.OperationName = operationName
				onError(&r.errOprNotFound)
				return
			}
			oprIndex = i
			break
		}
		if len(operationName) > 0 &&
			string(operationName) == string(x.Value) {
			// Found selected operation
			oprIndex = i
		}
		for j := i + 1; j < len(r.operations); j++ {
			x2 := r.buffer[r.operations[j].IndexStart+1]
			if x2.ID != gqlscan.TokenOprName {
				continue
			}
			if string(x.Value) == string(x2.Value) {
				// Found redeclared operation
				r.errRedecOpr.OperationName = x.Value
				onError(&r.errRedecOpr)
				return
			}
		}
	}

	// Make sure the selected operation was found
	if oprIndex < 0 {
		r.errOprNotFound.OperationName = operationName
		onError(&r.errOprNotFound)
		return
	}

	// Initialize operation buffer
	i := 1
	opr := r.operations[oprIndex]
	o := r.buffer[opr.IndexStart:opr.IndexEnd]
	r.bufferOpr = append(r.bufferOpr, o[0])
	if o[1].ID == gqlscan.TokenVarList ||
		o[1].ID == gqlscan.TokenOprName &&
			o[2].ID == gqlscan.TokenVarList {
		// Has variable list
		if endOfVarBlock, ok := r.validateAndIndexVars(
			o, varsJSON, onError,
		); !ok {
			return
		} else {
			i = endOfVarBlock + 1
		}

		if o[1].ID == gqlscan.TokenOprName {
			// Include operation name
			r.bufferOpr = append(r.bufferOpr, o[1])
		}
	}

	// Validate variable JSON
	if len(varsJSON) > 0 {
		if !gjson.ValidBytes(varsJSON) {
			onError(&r.errVarJSONSyntax)
			return
		}

		v := gjson.Parse(unsafe.B2S(varsJSON))
		if !v.IsObject() {
			r.errVarJSONNotObj.Received = v
			onError(&r.errVarJSONNotObj)
			return
		}
	}

	// Construct variable values
	for _, varName := range r.varOrder {
		vr, _ := r.varsDecls.Get(varName)
		if res := gjson.Get(
			unsafe.B2S(varsJSON), unsafe.B2S(vr.Name),
		); res.Exists() {
			r.writeTypeToStack(r.buffer, vr.Type.IndexStart)
			r.buffer2 = r.buffer2[:0]
			if r.writeValueToBuffer(0, false, res) {
				r.errUnexpValType.Buffer = r.buffer
				r.errUnexpValType.BufferJSON = varsJSON
				r.errUnexpValType.TypeExpected = vr.Type
				r.errUnexpValType.DefaultValueReceived = vr.DefaultValue
				r.errUnexpValType.JSONValueReceived = indexRange{
					IndexStart: res.Index,
					IndexEnd:   res.Index + len(res.Raw),
				}
				isErr = true
				onError(&r.errUnexpValType)
				return
			}

			r.varsConstructed.Append(r.buffer2...)

		} else if vr.DefaultValue.IndexStart > -1 {
			// Use default value when no value is present in JSON
			v := r.buffer[vr.DefaultValue.IndexStart:vr.DefaultValue.IndexEnd]
			r.varsConstructed.Append(v...)

		} else {
			isErr = true
			r.errVarUndefined.VariableName = vr.Name
			onError(&r.errVarUndefined)
			return
		}
		seg := r.varsConstructed.Cut(varName)
		val := r.varsConstructed.GetSegment(seg)
		r.varValues = append(r.varValues, val)
	}
	if isErr {
		return
	}

	// Make sure there are no recursive fragments
	r.errFragRecurse.Path = r.errFragRecurse.Path[:0]
	r.gi.Make(
		r.fragmentGraphEdges,
		func(nodeName []byte) {
			r.errFragRecurse.Path = append(r.errFragRecurse.Path, nodeName)
		},
		func(fragName []byte) {
			r.ordered = append(r.ordered, fragName)
		},
	)
	if len(r.errFragRecurse.Path) > 0 {
		onError(&r.errFragRecurse)
		return
	}

	// Construct nested fragments in the order of dependency
	for _, fragName := range r.ordered {
		if !r.fragDefs.GetFn(fragName, func(fd *fragDef) {
			// Flag fragment definition as used
			fd.Used = true

			if r.constructFrag(fragName, fd.indexRange, onError) {
				return
			}
		}) {
			r.errFragUndefined.FragmentName = fragName
			onError(&r.errFragUndefined)
			return
		}
	}

	// Construct and mark all used entry fragments
	r.entryFrags.VisitAll(func(fragName []byte, value struct{}) {
		ok := r.fragDefs.GetFn(fragName, func(v *fragDef) {
			v.Used = true
			if c := r.fragsConstructed.GetItems(fragName); c != nil {
				return
			}
			if r.constructFrag(fragName, v.indexRange, onError) {
				isErr = true
				return
			}
		})
		if !ok {
			isErr = true
			r.errFragUndefined.FragmentName = fragName
			onError(&r.errFragUndefined)
			return
		}
	})
	if isErr {
		return
	}

	// Define variable list if necessary
	if len(r.varOrder) > 0 {
		r.bufferOpr = append(r.bufferOpr, Token{
			ID: gqlscan.TokenVarList,
		})
		for _, name := range r.varOrder {
			// Write variable name
			r.bufferOpr = append(r.bufferOpr, Token{
				ID:    gqlscan.TokenVarName,
				Value: name,
			})
			// Write variable type
			r.varsDecls.GetFn(name, func(v *varDecl) {
				typeDef := r.buffer[v.Type.IndexStart:v.Type.IndexEnd]
				r.bufferOpr = append(r.bufferOpr, typeDef...)
			})
			// Write variable value
			val := r.varsConstructed.GetItems(name)
			r.bufferOpr = append(r.bufferOpr, val...)
		}
		r.bufferOpr = append(r.bufferOpr, Token{
			ID: gqlscan.TokenVarListEnd,
		})
	}

	// Fill operation buffer
	selectionSetIndex := len(r.bufferOpr)
	for ; i < len(o); i++ {
		switch o[i].ID {
		default:
			r.bufferOpr = append(r.bufferOpr, o[i])

		case gqlscan.TokenNamedSpread:
			// Inline fragment spread outside of fragment definitions
			var fragContents []Token
			r.fragDefs.GetFn(o[i].Value, func(v *fragDef) {
				fragContents = r.buffer[v.IndexStart+3 : v.IndexEnd-1]
				v.Used = true
			})

			if v := r.fragsConstructed.GetItems(o[i].Value); v != nil {
				r.bufferOpr = append(r.bufferOpr, v...)
			} else {
				r.bufferOpr = append(r.bufferOpr, fragContents...)
			}

		case gqlscan.TokenVarRef:
			// Append variable value index
			value := r.varsConstructed.Get(o[i].Value)
			if value.Index == -1 {
				r.errVarUndeclared.VariableName = o[i].Value
				onError(&r.errVarUndeclared)
				return
			}
			r.bufferOpr = append(
				r.bufferOpr,
				makeVarValueRefTok(value.Index, o[i].Value),
			)
		}
	}

	// Make sure all defined fragments were used
	r.fragDefs.Visit(func(key []byte, value fragDef) (stop bool) {
		if value.Used {
			return false
		}
		isErr = true
		r.errFragUnused.FragmentName = r.buffer[value.IndexStart].Value
		onError(&r.errFragUnused)
		return true
	})
	if isErr {
		return
	}

	onSuccess(r.varValues, r.bufferOpr, r.bufferOpr[selectionSetIndex:])
}

type ErrorSyntax struct {
	ScanErr gqlscan.Error
}

func (e *ErrorSyntax) Error() string {
	return fmt.Sprintf("syntax error: %s", e.ScanErr.Error())
}

type ErrorOprAnonNonExcl struct{}

func (e *ErrorOprAnonNonExcl) Error() string {
	return "non-exclusive anonymous operation"
}

type ErrorOprNotFound struct {
	OperationName []byte
}

func (e *ErrorOprNotFound) Error() string {
	return fmt.Sprintf("operation %q not found", e.OperationName)
}

type ErrorRedecOpr struct {
	OperationName []byte
}

func (e *ErrorRedecOpr) Error() string {
	return fmt.Sprintf("operation %q redeclared", e.OperationName)
}

type ErrorRedecFrag struct {
	FragmentName []byte
}

func (e *ErrorRedecFrag) Error() string {
	return fmt.Sprintf("fragment %q redeclared", e.FragmentName)
}

type ErrorFragUnused struct {
	FragmentName []byte
}

func (e *ErrorFragUnused) Error() string {
	return fmt.Sprintf("fragment %q unused", e.FragmentName)
}

type ErrorFragUndefined struct {
	FragmentName []byte
}

func (e *ErrorFragUndefined) Error() string {
	return fmt.Sprintf("fragment %q undefined", e.FragmentName)
}

type ErrorFragRecurse struct {
	Path [][]byte
}

func (e *ErrorFragRecurse) Error() string {
	const msg = "fragment recursion detected at: "
	var b strings.Builder
	tl := len(msg) + len(e.Path) - 1
	for i := range e.Path {
		tl += len(e.Path[i])
	}
	b.Grow(tl)
	_, _ = b.WriteString(msg)
	for i := range e.Path {
		tl += len(e.Path[i])
		_, _ = b.Write(e.Path[i])
		if i+1 < len(e.Path) {
			_, _ = b.WriteString(".")
		}
	}
	return b.String()
}

type ErrorFragLimitExceeded struct {
	Limit int
}

func (e *ErrorFragLimitExceeded) Error() string {
	return fmt.Sprintf("fragment limit (%d) exceeded", e.Limit)
}

type ErrorRedeclVar struct {
	VariableName []byte
}

func (e *ErrorRedeclVar) Error() string {
	return fmt.Sprintf("variable %q redeclared", e.VariableName)
}

type ErrorVarUndeclared struct {
	VariableName []byte
}

func (e *ErrorVarUndeclared) Error() string {
	return fmt.Sprintf("variable %q undeclared", e.VariableName)
}

type ErrorVarUndefined struct {
	VariableName []byte
}

func (e *ErrorVarUndefined) Error() string {
	return fmt.Sprintf("variable %q undefined", e.VariableName)
}

type ErrorUnexpValType struct {
	Buffer               []Token
	BufferJSON           []byte
	TypeExpected         indexRange
	DefaultValueReceived indexRange
	JSONValueReceived    indexRange
}

func (e *ErrorUnexpValType) Error() string {
	var b strings.Builder
	b.WriteString("unexpected value type, expected: ")
	WriteTypeDesignation(
		&b,
		e.Buffer[e.TypeExpected.IndexStart:e.TypeExpected.IndexEnd],
	)
	dvr := e.DefaultValueReceived
	if dvr.IndexStart > -1 {
		b.WriteString("; received(default): ")
		WriteValue(&b, e.Buffer[dvr.IndexStart:dvr.IndexEnd])
	}
	jvr := e.JSONValueReceived
	if jvr.IndexStart > -1 {
		b.WriteString("; received(json): ")
		b.Write(e.BufferJSON[jvr.IndexStart:jvr.IndexEnd])
	}
	return b.String()
}

type ErrorVarJSONSyntax struct{}

func (e *ErrorVarJSONSyntax) Error() string {
	return "variables JSON syntax error"
}

type ErrorVarJSONNotObj struct {
	Received gjson.Result
}

func (e *ErrorVarJSONNotObj) Error() string {
	return fmt.Sprintf(
		"expected JSON object for variables, received: %s",
		e.Received.String(),
	)
}

func (r *Parser) validateAndIndexVars(
	t []Token, varsJSON []byte, onError func(error),
) (indexEndVarList int, ok bool) {
	// Validate and index all variable declarations
	i := 2
	if t[1].ID == gqlscan.TokenOprName {
		i = 3
	}
	for {
		if t[i].ID == gqlscan.TokenVarListEnd {
			indexEndVarList = i
			break
		}

		name := t[i].Value
		i++

		typeIndex := r.writeTypeToStack(t, i)
		i = typeIndex.IndexEnd

		// Check optional default value
		defVal := indexRange{IndexStart: -1}
		isErr := false
		if t[i].ID != gqlscan.TokenVarName &&
			t[i].ID != gqlscan.TokenVarListEnd {
			defVal.IndexStart = i
			// Parse default value
			arrayLevel := 0
			expect := r.typeStack.Get(0)
		LOOP:
			for {
				switch t[i].ID {
				case gqlscan.TokenArr:
					i++
					if !expect.Array {
						isErr = true
					}
					arrayLevel++
					if arrayLevel < r.typeStack.Len() {
						expect = r.typeStack.Get(arrayLevel)
					} else {
						isErr = true
					}
				case gqlscan.TokenArrEnd:
					i++
					arrayLevel--
					if arrayLevel > -1 {
						expect = r.typeStack.Get(arrayLevel)
					}
				case gqlscan.TokenNull:
					i++
					if !expect.Nullable {
						isErr = true
					}
				case gqlscan.TokenInt:
					i++
					if string(expect.TypeName) != "Int" {
						isErr = true
					}
				case gqlscan.TokenFloat:
					i++
					if string(expect.TypeName) != "Float" {
						isErr = true
					}
				case gqlscan.TokenTrue, gqlscan.TokenFalse:
					i++
					if string(expect.TypeName) != "Boolean" {
						isErr = true
					}
				case gqlscan.TokenStr, gqlscan.TokenStrBlock:
					i++
					if string(expect.TypeName) != "String" &&
						string(expect.TypeName) != "ID" {
						isErr = true
					}
				case gqlscan.TokenObj:
					i++
				SKIP_OBJ_INTERNALS:
					for lvl := 0; ; i++ {
						switch t[i].ID {
						case gqlscan.TokenObj:
							lvl++
						case gqlscan.TokenObjEnd:
							lvl--
							if lvl < 0 {
								i++
								break SKIP_OBJ_INTERNALS
							}
						}
					}
					if string(expect.TypeName) == "Int" ||
						string(expect.TypeName) == "Float" ||
						string(expect.TypeName) == "Boolean" ||
						string(expect.TypeName) == "String" ||
						string(expect.TypeName) == "ID" {
						isErr = true
					}
				}
				if arrayLevel < 1 {
					break LOOP
				}
			}
			defVal.IndexEnd = i
		}

		if isErr {
			r.errUnexpValType.Buffer = r.buffer
			r.errUnexpValType.BufferJSON = varsJSON
			r.errUnexpValType.TypeExpected = typeIndex
			r.errUnexpValType.DefaultValueReceived = defVal
			r.errUnexpValType.JSONValueReceived = indexRange{IndexStart: -1}
			onError(&r.errUnexpValType)
			return
		}

		r.varOrder = append(r.varOrder, name)
		r.varsDecls.SetFn(name, func(v *varDecl) varDecl {
			if v != nil {
				isErr = true
				r.errRedeclVar.VariableName = name
				onError(&r.errRedeclVar)
				return varDecl{}
			}
			return varDecl{
				Name:         name,
				Type:         typeIndex,
				DefaultValue: defVal,
			}
		})
		if isErr {
			return 0, false
		}
	}
	return indexEndVarList, true
}

// WriteTypeDesignation stringifies a type designation to w
// expecting definition to be valid.
func WriteTypeDesignation(w io.Writer, definition []Token) {
	for i := 0; i < len(definition); i++ {
		switch definition[i].ID {
		case gqlscan.TokenVarTypeName:
			_, _ = w.Write(definition[i].Value)
		case gqlscan.TokenVarTypeArr:
			_, _ = w.Write(strSqrBrackLeft)
		case gqlscan.TokenVarTypeArrEnd:
			_, _ = w.Write(strSqrBrackRight)
		case gqlscan.TokenVarTypeNotNull:
			_, _ = w.Write(strNotNull)
		}
	}
}

// WriteValue stringifies a value to w expecting definition to be valid.
func WriteValue(w io.Writer, definition []Token) {
	level := 0
	for i := 0; i < len(definition); {
		switch definition[i].ID {
		case gqlscan.TokenInt, gqlscan.TokenFloat:
			_, _ = w.Write(definition[i].Value)
		case gqlscan.TokenStr:
			_, _ = w.Write([]byte(fmt.Sprintf("%q", definition[i].Value)))
		case gqlscan.TokenFalse:
			_, _ = w.Write(strFalse)
		case gqlscan.TokenTrue:
			_, _ = w.Write(strTrue)
		case gqlscan.TokenNull:
			_, _ = w.Write(strNull)
		case gqlscan.TokenArr:
			_, _ = w.Write(strSqrBrackLeft)
			level++
		case gqlscan.TokenArrEnd:
			_, _ = w.Write(strSqrBrackRight)
			level--
		case gqlscan.TokenObj:
			_, _ = w.Write(strCurlBrackLeft)
			level++
		case gqlscan.TokenObjEnd:
			_, _ = w.Write(strCurlBrackRight)
			level--
		case gqlscan.TokenObjField:
			_, _ = w.Write(definition[i].Value)
			_, _ = w.Write(strColSp)
			i++
			continue
		}
		i++
		if i < len(definition) &&
			definition[i-1].ID != gqlscan.TokenObj &&
			definition[i-1].ID != gqlscan.TokenArr &&
			definition[i].ID != gqlscan.TokenObjEnd &&
			definition[i].ID != gqlscan.TokenArrEnd {
			_, _ = w.Write(strComSp)
		}
	}
}

var (
	strNotNull        = []byte("!")
	strSqrBrackLeft   = []byte("[")
	strSqrBrackRight  = []byte("]")
	strCurlBrackLeft  = []byte("{")
	strCurlBrackRight = []byte("}")
	strColSp          = []byte(":")
	strComSp          = []byte(",")
	strTrue           = []byte("true")
	strFalse          = []byte("false")
	strNull           = []byte("null")
)

func (r *Parser) writeTypeToStack(t []Token, i int) (typeIndex indexRange) {
	r.typeStack.Reset()
	typeIndex.IndexStart = i
LOOP:
	for offset := 0; ; i++ {
		switch t[i].ID {
		case gqlscan.TokenVarTypeArr:
			r.typeStack.Push(typeUnion{
				Nullable: true,
				Array:    true,
			})
		case gqlscan.TokenVarTypeArrEnd:
			offset++
		case gqlscan.TokenVarTypeName:
			r.typeStack.Push(typeUnion{
				Nullable: true,
				TypeName: t[i].Value,
			})
		case gqlscan.TokenVarTypeNotNull:
			r.typeStack.TopOffsetFn(offset, func(t *typeUnion) {
				t.Nullable = false
			})
		default:
			break LOOP
		}
	}
	typeIndex.IndexEnd = i
	return typeIndex
}

func (r *Parser) writeValueToBuffer(
	arrayLevel int,
	isErr bool,
	v gjson.Result,
) bool {
	var expect typeUnion
	if arrayLevel > -1 && arrayLevel < r.typeStack.Len() {
		expect = r.typeStack.Get(arrayLevel)
	}
	switch {
	case v.Type == gjson.Null:
		if arrayLevel > -1 && !expect.Nullable {
			isErr = true
		}
		r.buffer2 = append(r.buffer2, Token{ID: gqlscan.TokenNull})

	case v.Type == gjson.Number:
		if strings.IndexByte(v.Raw, '.') > -1 {
			// Not an integer
			if arrayLevel > -1 && string(expect.TypeName) != "Float" {
				isErr = true
			}
			r.buffer2 = append(r.buffer2, Token{
				ID:    gqlscan.TokenFloat,
				Value: unsafe.S2B(v.Raw),
			})
		} else {
			// Integer
			if arrayLevel > -1 &&
				string(expect.TypeName) != "Float" &&
				string(expect.TypeName) != "Int" {
				isErr = true
			}
			r.buffer2 = append(r.buffer2, Token{
				ID:    gqlscan.TokenInt,
				Value: unsafe.S2B(v.Raw),
			})
		}

	case v.Type == gjson.True:
		if arrayLevel > -1 && string(expect.TypeName) != "Boolean" {
			isErr = true
		}
		r.buffer2 = append(r.buffer2, Token{ID: gqlscan.TokenTrue})

	case v.Type == gjson.False:
		if arrayLevel > -1 && string(expect.TypeName) != "Boolean" {
			isErr = true
		}
		r.buffer2 = append(r.buffer2, Token{ID: gqlscan.TokenFalse})

	case v.Type == gjson.String:
		if arrayLevel > -1 &&
			(expect.Array ||
				string(expect.TypeName) == "Int" ||
				string(expect.TypeName) == "Float" ||
				string(expect.TypeName) == "Boolean") {
			isErr = true
		}
		switch string(expect.TypeName) {
		case "", "String", "ID":
			r.buffer2 = append(r.buffer2, Token{
				ID:    gqlscan.TokenStr,
				Value: unsafe.S2B(v.Raw[1 : len(v.Raw)-1]),
			})
		default:
			r.buffer2 = append(r.buffer2, Token{
				ID:    gqlscan.TokenEnumVal,
				Value: unsafe.S2B(v.Raw[1 : len(v.Raw)-1]),
			})
		}

	case v.IsArray():
		if arrayLevel > -1 && !expect.Array {
			isErr = true
		}
		r.buffer2 = append(r.buffer2, Token{ID: gqlscan.TokenArr})
		al := arrayLevel
		if al > -1 {
			al++
		}
		v.ForEach(func(key, value gjson.Result) bool {
			isErr = r.writeValueToBuffer(al, isErr, value)
			return true
		})
		r.buffer2 = append(r.buffer2, Token{ID: gqlscan.TokenArrEnd})

	case v.IsObject():
		if arrayLevel > -1 && (string(expect.TypeName) == "Int" ||
			string(expect.TypeName) == "Float" ||
			string(expect.TypeName) == "Boolean" ||
			string(expect.TypeName) == "String" ||
			string(expect.TypeName) == "ID") {
			isErr = true
		}
		r.buffer2 = append(r.buffer2, Token{ID: gqlscan.TokenObj})
		v.ForEach(func(key, value gjson.Result) bool {
			r.buffer2 = append(r.buffer2, Token{
				ID:    gqlscan.TokenObjField,
				Value: unsafe.S2B(key.Raw[1 : len(key.Raw)-1]),
			})
			isErr = r.writeValueToBuffer(-1, isErr, value)
			return true
		})
		r.buffer2 = append(r.buffer2, Token{ID: gqlscan.TokenObjEnd})
	}
	return isErr
}

func (r *Parser) constructFrag(
	fragName []byte,
	rn indexRange,
	onError func(err error),
) (err bool) {
	fragSelections := r.buffer[rn.IndexStart+3 : rn.IndexEnd-1]
	for _, t := range fragSelections {
		switch t.ID {
		case gqlscan.TokenNamedSpread:
			// Inline fragment spread
			fragContents := r.fragsConstructed.GetItems(t.Value)
			r.fragsConstructed.Append(fragContents...)
		case gqlscan.TokenVarRef:
			valueSeg := r.varsConstructed.Get(t.Value)
			if valueSeg.Index == -1 {
				r.errVarUndefined.VariableName = t.Value
				onError(&r.errVarUndefined)
				return true
			}
			// value := r.varsConstructed.GetSegment(valueSeg)
			r.fragsConstructed.Append(
				makeVarValueRefTok(valueSeg.Index, t.Value),
			)
		default:
			r.fragsConstructed.Append(t)
		}
	}
	r.fragsConstructed.Cut(fragName)
	return false
}

func makeVarValueRefTok(index int, variableName []byte) Token {
	return Token{
		ID:    gqlscan.Token(TokenTypeValIndexOffset + index),
		Value: variableName,
	}
}
