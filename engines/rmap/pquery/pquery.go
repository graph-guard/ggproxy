package pquery

import (
	"fmt"
	"io"
	"strconv"

	"github.com/graph-guard/ggproxy/gqlparse"
	"github.com/graph-guard/ggproxy/utilities/container/amap"
	"github.com/graph-guard/ggproxy/utilities/container/hamap"
	"github.com/graph-guard/ggproxy/utilities/stack"
	"github.com/graph-guard/ggproxy/utilities/unsafe"
	"github.com/graph-guard/ggproxy/utilities/xxhash"
	"github.com/graph-guard/gqlscan"
)

type pathTerminal struct{}
type selectTerminal struct{}
type argumentsTerminal struct{}
type argumentPathTerminal struct{}
type objectTerminal struct{}

// QueryPart is a structure of query pash hash and value,
// when united represents a parted query.
// Used for a template fast search by rmap.
type QueryPart struct {
	ArgLeafIdx int
	Hash       uint64
	Value      any
}

// Maker is a meta structure to store the runtime data and the hash seed.
type Maker struct {
	mstack    *stack.Stack[any]
	pstack    *stack.Stack[xxhash.Hash]
	qmap      *amap.Map[uint64, bool]
	usedStack *stack.Stack[any]
	arrayPool *stack.Stack[*[]any]
	mapPool   *stack.Stack[*hamap.Map[string, any]]
	seed      uint64
}

// NewMaker creates a new instance of Maker.
// Accepts a hash seed.
func NewMaker(seed uint64) *Maker {
	return &Maker{
		mstack:    stack.New[any](256),
		pstack:    stack.New[xxhash.Hash](256),
		qmap:      amap.New[uint64, bool](256),
		mapPool:   stack.New[*hamap.Map[string, any]](128),
		arrayPool: stack.New[*[]any](128),
		usedStack: stack.New[any](128),
		seed:      seed,
	}
}

// ParseQuery parses query into QueryParts.
// Accepts a token list.
// QueryParts are accessible through the fn function.
func (m *Maker) ParseQuery(
	variableValues [][]gqlparse.Token,
	queryType gqlscan.Token,
	selectionSet []gqlparse.Token,
	fn func(qp QueryPart) (stop bool),
) {
	m.mstack.Reset()
	m.pstack.Reset()
	m.qmap.Reset()

	var pathHash uint64
	var insideArray, argLeafIdx int = 0, -1
	var lastObjField string

	switch queryType {
	case gqlscan.TokenDefQry:
		path := xxhash.New(m.seed)
		xxhash.Write(&path, "query")
		m.pstack.Push(path)
	case gqlscan.TokenDefMut:
		path := xxhash.New(m.seed)
		xxhash.Write(&path, "mutation")
		m.pstack.Push(path)
	default:
		panic(fmt.Errorf("unsupported query type: %v", queryType))
	}

	for _, token := range selectionSet {
		if ix := token.VariableIndex(); ix > -1 {
			value := variableValues[ix]
			for _, token := range value {
				switch token.ID {
				case gqlscan.TokenArr:
					insideArray++
					var arr *[]any
					if m.arrayPool.Len() > 0 {
						arr = m.arrayPool.Pop()
					} else {
						arr = &[]any{}
					}
					m.usedStack.Push(arr)

					switch t := m.mstack.Top(); t := t.(type) {
					case *[]any:
						*t = append(*t, arr)
					case *hamap.Map[string, any]:
						t.Set(lastObjField, arr)
					}
					m.mstack.Push(arr)
				case gqlscan.TokenObj:
					if insideArray == 0 {
						path := m.pstack.Top()
						xxhash.Write(&path, ".")
						m.pstack.Push(path)
						m.mstack.Push(objectTerminal{})
					} else {
						var obj *hamap.Map[string, any]
						if m.mapPool.Len() > 0 {
							obj = m.mapPool.Pop()
						} else {
							obj = hamap.New[string, any](64, nil)
						}
						m.usedStack.Push(obj)

						switch t := m.mstack.Top(); t := t.(type) {
						case *[]any:
							*t = append(*t, obj)
						case *hamap.Map[string, any]:
							t.Set(lastObjField, obj)
						}
						m.mstack.Push(obj)
					}
				case gqlscan.TokenObjField:
					if insideArray == 0 {
						t := m.pstack.Top()
						xxhash.Write(&t, token.Value)
						m.pstack.Push(t)
						m.mstack.Push(pathTerminal{})
					} else {
						lastObjField = unsafe.B2S(token.Value)
					}
				case gqlscan.TokenStr, gqlscan.TokenEnumVal, gqlscan.TokenInt,
					gqlscan.TokenFloat, gqlscan.TokenTrue, gqlscan.TokenFalse, gqlscan.TokenNull:
					var val any
					var err error
					switch token.ID {
					case gqlscan.TokenStr, gqlscan.TokenEnumVal:
						val = token.Value
					case gqlscan.TokenInt:
						val, err = strconv.ParseInt(unsafe.B2S(token.Value), 10, 64)
						if err != nil {
							panic(err)
						}
					case gqlscan.TokenFloat:
						val, err = strconv.ParseFloat(unsafe.B2S(token.Value), 64)
						if err != nil {
							panic(err)
						}
					case gqlscan.TokenTrue:
						val = true
					case gqlscan.TokenFalse:
						val = false
					}
					if insideArray == 0 {
						switch t := m.mstack.Top(); t.(type) {
						case pathTerminal:
							m.mstack.Pop()
							path := m.pstack.Pop()
							pathHash = path.Sum64()
							if _, ok := m.qmap.Get(pathHash); !ok {
								argLeafIdx++
								m.qmap.Set(pathHash, true)
								if fn(QueryPart{ArgLeafIdx: argLeafIdx, Hash: pathHash, Value: val}) {
									return
								}
							}
						}
					} else {
						switch t := m.mstack.Top(); t := t.(type) {
						case *[]any:
							*t = append(*t, val)
						case *hamap.Map[string, any]:
							t.Set(lastObjField, val)
						}
					}
				case gqlscan.TokenArrEnd, gqlscan.TokenObjEnd:
					if token.ID == gqlscan.TokenArrEnd {
						insideArray--
					}
					for {
						switch t := m.mstack.Top(); t.(type) {
						case pathTerminal:
							m.mstack.Pop()
							path := m.pstack.Pop()
							pathHash = path.Sum64()
							if _, ok := m.qmap.Get(pathHash); !ok {
								m.qmap.Set(pathHash, true)
								if fn(QueryPart{ArgLeafIdx: -1, Hash: pathHash, Value: nil}) {
									return
								}
							}
							continue
						case argumentPathTerminal:
							m.mstack.Pop()
							m.pstack.Pop()
							continue
						case argumentsTerminal:
							m.mstack.Pop()
							m.pstack.Pop()
						case selectTerminal, objectTerminal:
							m.mstack.Pop()
							m.pstack.Pop()
							m.mstack.Pop()
							m.pstack.Pop()
						case *[]any, *hamap.Map[string, any]:
							el := m.mstack.Pop()
							path := m.pstack.Top()
							if insideArray == 0 {
								pathHash = path.Sum64()
								switch elt := el.(type) {
								case *[]any, *hamap.Map[string, any]:
									if _, ok := m.qmap.Get(pathHash); !ok {
										argLeafIdx++
										m.qmap.Set(pathHash, true)
										if fn(QueryPart{ArgLeafIdx: argLeafIdx, Hash: pathHash, Value: elt}) {
											return
										}
									}
								}
								m.mstack.Pop()
								m.pstack.Pop()
							}
						}
						break
					}
				}
			}
			continue
		}

		switch token.ID {
		case gqlscan.TokenField, gqlscan.TokenFragInline:
			switch t := m.mstack.Top(); t.(type) {
			case argumentPathTerminal:
				m.mstack.Pop()
				m.pstack.Pop()
			}
			switch t := m.mstack.Top(); t.(type) {
			case pathTerminal:
				m.mstack.Pop()
				path := m.pstack.Pop()
				pathHash = path.Sum64()
				if _, ok := m.qmap.Get(pathHash); !ok {
					m.qmap.Set(pathHash, true)
					if fn(QueryPart{ArgLeafIdx: -1, Hash: pathHash, Value: nil}) {
						return
					}
				}
			}
			path := m.pstack.Top()
			if token.ID == gqlscan.TokenFragInline {
				xxhash.Write(&path, "|")
			}
			xxhash.Write(&path, token.Value)
			m.pstack.Push(path)
			m.mstack.Push(pathTerminal{})
		case gqlscan.TokenArgList:
			m.mstack.PopPush(argumentPathTerminal{})
			path := m.pstack.Top()
			xxhash.Write(&path, ".")
			m.pstack.Push(path)
			m.mstack.Push(argumentsTerminal{})
		case gqlscan.TokenSet:
			path := m.pstack.Top()
			xxhash.Write(&path, ".")
			m.pstack.Push(path)
			m.mstack.Push(selectTerminal{})
		case gqlscan.TokenArgName:
			path := m.pstack.Top()
			xxhash.Write(&path, token.Value)
			m.pstack.Push(path)
			m.mstack.Push(pathTerminal{})
		case gqlscan.TokenArr:
			insideArray++
			var arr *[]any
			if m.arrayPool.Len() > 0 {
				arr = m.arrayPool.Pop()
			} else {
				arr = &[]any{}
			}
			m.usedStack.Push(arr)

			switch t := m.mstack.Top(); t := t.(type) {
			case *[]any:
				*t = append(*t, arr)
			case *hamap.Map[string, any]:
				t.Set(lastObjField, arr)
			}
			m.mstack.Push(arr)
		case gqlscan.TokenObj:
			if insideArray == 0 {
				path := m.pstack.Top()
				xxhash.Write(&path, ".")
				m.pstack.Push(path)
				m.mstack.Push(objectTerminal{})
			} else {
				var obj *hamap.Map[string, any]
				if m.mapPool.Len() > 0 {
					obj = m.mapPool.Pop()
				} else {
					obj = hamap.New[string, any](64, nil)
				}
				m.usedStack.Push(obj)

				switch t := m.mstack.Top(); t := t.(type) {
				case *[]any:
					*t = append(*t, obj)
				case *hamap.Map[string, any]:
					t.Set(lastObjField, obj)
				}
				m.mstack.Push(obj)
			}
		case gqlscan.TokenObjField:
			if insideArray == 0 {
				t := m.pstack.Top()
				xxhash.Write(&t, token.Value)
				m.pstack.Push(t)
				m.mstack.Push(pathTerminal{})
			} else {
				lastObjField = unsafe.B2S(token.Value)
			}
		case gqlscan.TokenStr, gqlscan.TokenEnumVal, gqlscan.TokenInt,
			gqlscan.TokenFloat, gqlscan.TokenTrue, gqlscan.TokenFalse, gqlscan.TokenNull:
			var val any
			var err error
			switch token.ID {
			case gqlscan.TokenStr, gqlscan.TokenEnumVal:
				val = token.Value
			case gqlscan.TokenNull:
				val = nil
			case gqlscan.TokenInt:
				val, err = strconv.ParseInt(unsafe.B2S(token.Value), 10, 64)
				if err != nil {
					panic(err)
				}
			case gqlscan.TokenFloat:
				val, err = strconv.ParseFloat(unsafe.B2S(token.Value), 64)
				if err != nil {
					panic(err)
				}
			case gqlscan.TokenTrue:
				val = true
			case gqlscan.TokenFalse:
				val = false
			}
			if insideArray == 0 {
				switch t := m.mstack.Top(); t.(type) {
				case pathTerminal:
					m.mstack.Pop()
					path := m.pstack.Pop()
					pathHash = path.Sum64()
					if _, ok := m.qmap.Get(pathHash); !ok {
						argLeafIdx++
						m.qmap.Set(pathHash, true)
						if fn(QueryPart{ArgLeafIdx: argLeafIdx, Hash: pathHash, Value: val}) {
							return
						}
					}
				}
			} else {
				switch t := m.mstack.Top(); t := t.(type) {
				case *[]any:
					*t = append(*t, val)
				case *hamap.Map[string, any]:
					t.Set(lastObjField, val)
				}
			}
		case gqlscan.TokenArrEnd, gqlscan.TokenSetEnd,
			gqlscan.TokenArgListEnd, gqlscan.TokenObjEnd:
			if token.ID == gqlscan.TokenArgListEnd {
				argLeafIdx = -1
			}
			if token.ID == gqlscan.TokenArrEnd {
				insideArray--
			}
			for {
				switch t := m.mstack.Top(); t.(type) {
				case pathTerminal:
					m.mstack.Pop()
					path := m.pstack.Pop()
					pathHash = path.Sum64()
					if _, ok := m.qmap.Get(pathHash); !ok {
						m.qmap.Set(pathHash, true)
						if fn(QueryPart{ArgLeafIdx: -1, Hash: pathHash, Value: nil}) {
							return
						}
					}
					continue
				case argumentPathTerminal:
					m.mstack.Pop()
					m.pstack.Pop()
					continue
				case argumentsTerminal:
					m.mstack.Pop()
					m.pstack.Pop()
				case selectTerminal, objectTerminal:
					m.mstack.Pop()
					m.pstack.Pop()
					m.mstack.Pop()
					m.pstack.Pop()
				case *[]any, *hamap.Map[string, any]:
					el := m.mstack.Pop()
					path := m.pstack.Top()
					if insideArray == 0 {
						pathHash = path.Sum64()
						switch elt := el.(type) {
						case *[]any, *hamap.Map[string, any]:
							if _, ok := m.qmap.Get(pathHash); !ok {
								argLeafIdx++
								m.qmap.Set(pathHash, true)
								if fn(QueryPart{ArgLeafIdx: argLeafIdx, Hash: pathHash, Value: elt}) {
									return
								}
							}
						}
						m.mstack.Pop()
						m.pstack.Pop()
					}
				}
				break
			}
		}
	}

	for m.usedStack.Len() > 0 {
		switch el := m.usedStack.Pop().(type) {
		case *[]any:
			*el = (*el)[:0]
			m.arrayPool.Push(el)
		case *hamap.Map[string, any]:
			el.Reset()
			m.mapPool.Push(el)
		}
	}
}

// PrintNSpaces prints n spaces in a row
func PrintNSpaces(w io.Writer, n uint) {
	for i := uint(0); i < n; i++ {
		_, _ = w.Write([]byte(" "))
	}
}

// Print prints out the QueryPart
func (qp QueryPart) Print(w io.Writer) {
	qp.print(w, 0)
}

func (qp QueryPart) print(w io.Writer, indent uint) {
	PrintNSpaces(w, indent)
	fmt.Fprintf(w, "%d:", qp.Hash)
	switch vt := qp.Value.(type) {
	case *[]any:
		_, _ = w.Write([]byte("\n"))
		printArr(*vt, w, indent+2)
	case *hamap.Map[string, any]:
		_, _ = w.Write([]byte("\n"))
		printObj(*vt, w, indent+2)
	default:
		if qp.Value != nil {
			_, _ = w.Write([]byte(" "))
			if s, ok := qp.Value.([]byte); ok {
				fmt.Fprintln(w, string(s))
			} else {
				fmt.Fprintln(w, qp.Value)
			}
		} else {
			_, _ = w.Write([]byte("\n"))
		}
	}
}

func printArr(arr []any, w io.Writer, indent uint) {
	for _, v := range arr {
		PrintNSpaces(w, indent)
		_, _ = w.Write([]byte("-:\n"))
		switch vt := v.(type) {
		case *[]any:
			printArr(*vt, w, indent+2)
		case *hamap.Map[string, any]:
			printObj(*vt, w, indent+2)
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

func printObj(obj hamap.Map[string, any], w io.Writer, indent uint) {
	obj.Visit(func(key string, value any) (stop bool) {
		PrintNSpaces(w, indent)
		fmt.Fprintf(w, "%s:\n", key)
		switch vt := value.(type) {
		case *[]any:
			printArr(*vt, w, indent+2)
		case *hamap.Map[string, any]:
			printObj(*vt, w, indent+2)
		default:
			PrintNSpaces(w, indent+2)
			if s, ok := value.([]byte); ok {
				fmt.Fprintln(w, string(s))
			} else {
				fmt.Fprintln(w, value)
			}
		}
		return false
	})
}
