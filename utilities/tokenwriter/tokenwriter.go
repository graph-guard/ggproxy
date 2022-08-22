package tokenwriter

import (
	"fmt"
	"io"

	"github.com/graph-guard/ggproxy/gqlreduce"
	"github.com/graph-guard/gqlscan"
)

func Write(w io.Writer, t []gqlreduce.Token) (err error) {
	write := func(data []byte) (stop bool) {
		if _, err = w.Write(data); err != nil {
			return true
		}
		return false
	}

	for i := range t {
		switch t[i].Type {
		case gqlscan.TokenDefQry:
			if t[i+1].Type == gqlscan.TokenOprName {
				if write(partDefQry) {
					return
				}
			}
			// Write nothing, await following selection set
		case gqlscan.TokenDefMut:
			if write(partDefMut) {
				return
			}
		case gqlscan.TokenDefSub:
			if write(partDefSub) {
				return
			}
		case gqlscan.TokenDirName:
			if write(partDirName) {
				return
			}
			if write(t[i].Value) {
				return
			}
		case gqlscan.TokenArgList:
			if write(partParenthesisL) {
				return
			}
		case gqlscan.TokenArgListEnd:
			if write(partParenthesisR) {
				return
			}
		case gqlscan.TokenSet:
			if write(partCurlyBracketL) {
				return
			}
		case gqlscan.TokenSetEnd:
			if write(partCurlyBracketR) {
				return
			}
		case gqlscan.TokenFragInline:
			if t[i-1].Type == gqlscan.TokenField ||
				t[i-1].Type == gqlscan.TokenDirName ||
				t[i-1].Type == gqlscan.TokenSetEnd ||
				t[i-1].Type == gqlscan.TokenArgListEnd {
				if write(partSpace) {
					return
				}
			}
			if write(partFragInline) {
				return
			}
			if write(t[i].Value) {
				return
			}
		case gqlscan.TokenFieldAlias:
			if t[i-1].Type == gqlscan.TokenField ||
				t[i-1].Type == gqlscan.TokenDirName ||
				t[i-1].Type == gqlscan.TokenSetEnd ||
				t[i-1].Type == gqlscan.TokenArgListEnd {
				if write(partSpace) {
					return
				}
			}
			if write(t[i].Value) {
				return
			}
			if write(partColumn) {
				return
			}
		case gqlscan.TokenField:
			if t[i-1].Type == gqlscan.TokenField ||
				t[i-1].Type == gqlscan.TokenDirName ||
				t[i-1].Type == gqlscan.TokenSetEnd ||
				t[i-1].Type == gqlscan.TokenArgListEnd {
				if write(partSpace) {
					return
				}
			}
			if write(t[i].Value) {
				return
			}
		case gqlscan.TokenArgName:
			if t[i-1].Type != gqlscan.TokenArgList {
				if write(partSpace) {
					return
				}
			}
			if write(t[i].Value) {
				return
			}
			if write(partColumn) {
				return
			}
		case gqlscan.TokenEnumVal:
			if isTokenEndOfVal(t[i-1].Type) {
				if write(partSpace) {
					return
				}
			}
			if write(t[i].Value) {
				return
			}
		case gqlscan.TokenArr:
			if isTokenEndOfVal(t[i-1].Type) {
				if write(partSpace) {
					return
				}
			}
			if write(partSquareBracketL) {
				return
			}
		case gqlscan.TokenArrEnd:
			if write(partSquareBracketR) {
				return
			}
		case gqlscan.TokenStr:
			if isTokenEndOfVal(t[i-1].Type) {
				if write(partSpace) {
					return
				}
			}
			if write(partDoubleQuotes) {
				return
			}
			if write(t[i].Value) {
				return
			}
			if write(partDoubleQuotes) {
				return
			}
		case gqlscan.TokenStrBlock:
			if isTokenEndOfVal(t[i-1].Type) {
				if write(partSpace) {
					return
				}
			}
			if write(part3DoubleQuotes) {
				return
			}
			if write(t[i].Value) {
				return
			}
			if write(part3DoubleQuotes) {
				return
			}
		case gqlscan.TokenInt:
			if isTokenEndOfVal(t[i-1].Type) {
				if write(partSpace) {
					return
				}
			}
			if write(t[i].Value) {
				return
			}
		case gqlscan.TokenFloat:
			if isTokenEndOfVal(t[i-1].Type) {
				if write(partSpace) {
					return
				}
			}
			if write(t[i].Value) {
				return
			}
		case gqlscan.TokenTrue:
			if isTokenEndOfVal(t[i-1].Type) {
				if write(partSpace) {
					return
				}
			}
			if write(partTrue) {
				return
			}
		case gqlscan.TokenFalse:
			if isTokenEndOfVal(t[i-1].Type) {
				if write(partSpace) {
					return
				}
			}
			if write(partFalse) {
				return
			}
		case gqlscan.TokenNull:
			if isTokenEndOfVal(t[i-1].Type) {
				if write(partSpace) {
					return
				}
			}
			if write(partNull) {
				return
			}
		case gqlscan.TokenObj:
			if isTokenEndOfVal(t[i-1].Type) {
				if write(partSpace) {
					return
				}
			}
			if write(partCurlyBracketL) {
				return
			}
		case gqlscan.TokenObjEnd:
			if write(partCurlyBracketR) {
				return
			}
		case gqlscan.TokenObjField:
			if t[i-1].Type != gqlscan.TokenObj {
				if write(partSpace) {
					return
				}
			}
			if write(t[i].Value) {
				return
			}
			if write(partColumn) {
				return
			}
		case gqlscan.TokenOprName:
			if write(t[i].Value) {
				return
			}
			if write(partSpace) {
				return
			}
		default:
			return fmt.Errorf(
				"unsupported token type: %s",
				t[i].Type.String(),
			)
		}
	}
	return nil
}

var partSpace = []byte(" ")
var partDoubleQuotes = []byte("\"")
var part3DoubleQuotes = []byte("\"\"\"")
var partSquareBracketL = []byte("[")
var partSquareBracketR = []byte("]")
var partCurlyBracketL = []byte("{")
var partCurlyBracketR = []byte("}")
var partColumn = []byte(":")
var partDefQry = []byte("query ")
var partDefMut = []byte("mutation ")
var partDefSub = []byte("subscription ")
var partDirName = []byte(" @")
var partParenthesisL = []byte("(")
var partParenthesisR = []byte(")")
var partFragInline = []byte("...on ")
var partTrue = []byte("true")
var partFalse = []byte("false")
var partNull = []byte("null")

func isTokenEndOfVal(t gqlscan.Token) bool {
	switch t {
	case gqlscan.TokenEnumVal,
		gqlscan.TokenInt,
		gqlscan.TokenFloat,
		gqlscan.TokenStr,
		gqlscan.TokenStrBlock,
		gqlscan.TokenTrue,
		gqlscan.TokenFalse,
		gqlscan.TokenNull,
		gqlscan.TokenObjEnd,
		gqlscan.TokenArrEnd:
		return true
	}
	return false
}
