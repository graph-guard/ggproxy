package tokenwriter

import (
	"fmt"
	"io"

	"github.com/graph-guard/ggproxy/gqlparse"
	"github.com/graph-guard/gqlscan"
)

func Write(w io.Writer, tokens []gqlparse.Token) (err error) {
	write := func(data []byte) (stop bool) {
		if _, err = w.Write(data); err != nil {
			return true
		}
		return false
	}

	for i := range tokens {
		if tokens[i].VariableIndex() > -1 {
			if write(partVarDollar) {
				return
			}
			if write(tokens[i].Value) {
				return
			}
			continue
		}
		switch tokens[i].ID {
		case gqlscan.TokenVarList:
			if write(partParenthesisL) {
				return
			}
		case gqlscan.TokenVarName:
			if tokens[i-1].ID != gqlscan.TokenVarList {
				if write(partSpace) {
					return
				}
			}
			if write(partVarDollar) {
				return
			}
			if write(tokens[i].Value) {
				return
			}
			if write(partColumn) {
				return
			}
		case gqlscan.TokenVarTypeNotNull:
			if write(partExlMark) {
				return
			}
			if isTokenVal(tokens[i+1].ID) {
				if write(partEq) {
					return
				}
			}
		case gqlscan.TokenVarTypeArr:
			if write(partSquareBracketL) {
				return
			}
		case gqlscan.TokenVarTypeArrEnd:
			if write(partSquareBracketR) {
				return
			}
			if isTokenVal(tokens[i+1].ID) {
				if write(partEq) {
					return
				}
			}
		case gqlscan.TokenVarTypeName:
			if write(tokens[i].Value) {
				return
			}
			if isTokenVal(tokens[i+1].ID) {
				if write(partEq) {
					return
				}
			}
		case gqlscan.TokenVarListEnd:
			if write(partParenthesisR) {
				return
			}
		case gqlscan.TokenDefQry:
			if tokens[i+1].ID == gqlscan.TokenOprName {
				if write(partDefQry) {
					return
				}
				if write(partSpace) {
					return
				}
			} else if tokens[i+1].ID == gqlscan.TokenVarList {
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
			if write(tokens[i].Value) {
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
			if tokens[i-1].ID == gqlscan.TokenField ||
				tokens[i-1].ID == gqlscan.TokenDirName ||
				tokens[i-1].ID == gqlscan.TokenSetEnd ||
				tokens[i-1].ID == gqlscan.TokenArgListEnd {
				if write(partSpace) {
					return
				}
			}
			if write(partFragInline) {
				return
			}
			if write(tokens[i].Value) {
				return
			}
		case gqlscan.TokenFieldAlias:
			if tokens[i-1].ID == gqlscan.TokenField ||
				tokens[i-1].ID == gqlscan.TokenDirName ||
				tokens[i-1].ID == gqlscan.TokenSetEnd ||
				tokens[i-1].ID == gqlscan.TokenArgListEnd {
				if write(partSpace) {
					return
				}
			}
			if write(tokens[i].Value) {
				return
			}
			if write(partColumn) {
				return
			}
		case gqlscan.TokenField:
			if tokens[i-1].ID == gqlscan.TokenField ||
				tokens[i-1].ID == gqlscan.TokenDirName ||
				tokens[i-1].ID == gqlscan.TokenSetEnd ||
				tokens[i-1].ID == gqlscan.TokenArgListEnd {
				if write(partSpace) {
					return
				}
			}
			if write(tokens[i].Value) {
				return
			}
		case gqlscan.TokenArgName:
			if tokens[i-1].ID != gqlscan.TokenArgList {
				if write(partSpace) {
					return
				}
			}
			if write(tokens[i].Value) {
				return
			}
			if write(partColumn) {
				return
			}
		case gqlscan.TokenEnumVal:
			if isTokenEndOfVal(tokens[i-1].ID) {
				if write(partSpace) {
					return
				}
			}
			if write(tokens[i].Value) {
				return
			}
		case gqlscan.TokenArr:
			if isTokenEndOfVal(tokens[i-1].ID) {
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
			if isTokenEndOfVal(tokens[i-1].ID) {
				if write(partSpace) {
					return
				}
			}
			if write(partDoubleQuotes) {
				return
			}
			if write(tokens[i].Value) {
				return
			}
			if write(partDoubleQuotes) {
				return
			}
		case gqlscan.TokenStrBlock:
			if isTokenEndOfVal(tokens[i-1].ID) {
				if write(partSpace) {
					return
				}
			}
			if write(part3DoubleQuotes) {
				return
			}
			if write(tokens[i].Value) {
				return
			}
			if write(part3DoubleQuotes) {
				return
			}
		case gqlscan.TokenInt:
			if isTokenEndOfVal(tokens[i-1].ID) {
				if write(partSpace) {
					return
				}
			}
			if write(tokens[i].Value) {
				return
			}
		case gqlscan.TokenFloat:
			if isTokenEndOfVal(tokens[i-1].ID) {
				if write(partSpace) {
					return
				}
			}
			if write(tokens[i].Value) {
				return
			}
		case gqlscan.TokenTrue:
			if isTokenEndOfVal(tokens[i-1].ID) {
				if write(partSpace) {
					return
				}
			}
			if write(partTrue) {
				return
			}
		case gqlscan.TokenFalse:
			if isTokenEndOfVal(tokens[i-1].ID) {
				if write(partSpace) {
					return
				}
			}
			if write(partFalse) {
				return
			}
		case gqlscan.TokenNull:
			if isTokenEndOfVal(tokens[i-1].ID) {
				if write(partSpace) {
					return
				}
			}
			if write(partNull) {
				return
			}
		case gqlscan.TokenObj:
			if isTokenEndOfVal(tokens[i-1].ID) {
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
			if tokens[i-1].ID != gqlscan.TokenObj {
				if write(partSpace) {
					return
				}
			}
			if write(tokens[i].Value) {
				return
			}
			if write(partColumn) {
				return
			}
		case gqlscan.TokenOprName:
			if write(tokens[i].Value) {
				return
			}
			if write(partSpace) {
				return
			}
		default:
			return fmt.Errorf(
				"unsupported token type: %s",
				tokens[i].ID.String(),
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
var partExlMark = []byte("!")
var partEq = []byte("=")
var partDefQry = []byte("query")
var partDefMut = []byte("mutation ")
var partDefSub = []byte("subscription ")
var partDirName = []byte(" @")
var partParenthesisL = []byte("(")
var partParenthesisR = []byte(")")
var partFragInline = []byte("...on ")
var partTrue = []byte("true")
var partFalse = []byte("false")
var partNull = []byte("null")
var partVarDollar = []byte("$")

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

func isTokenVal(t gqlscan.Token) bool {
	switch t {
	case gqlscan.TokenEnumVal,
		gqlscan.TokenInt,
		gqlscan.TokenFloat,
		gqlscan.TokenStr,
		gqlscan.TokenStrBlock,
		gqlscan.TokenTrue,
		gqlscan.TokenFalse,
		gqlscan.TokenNull,
		gqlscan.TokenObj,
		gqlscan.TokenArr:
		return true
	}
	return false
}
