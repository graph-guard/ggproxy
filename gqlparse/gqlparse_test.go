package gqlparse_test

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/graph-guard/ggproxy/gqlparse"
	"github.com/graph-guard/ggproxy/utilities/decl"
	"github.com/graph-guard/ggproxy/utilities/testeq"
	"github.com/graph-guard/gqlscan"
	"github.com/stretchr/testify/require"
)

var testdata = []decl.Declaration[TestSuccess]{
	decl.New(TestSuccess{
		Src:      "query($e: Episode!) {hero(episode: $e) {id}}",
		VarsJSON: `{"e": "EMPIRE"}`,
		ExpectVarVals: map[string][]gqlparse.Token{
			"e": {Token(gqlscan.TokenEnumVal, "EMPIRE")},
		},
		ExpectOpr: []gqlparse.Token{
			Token(gqlscan.TokenDefQry),
			Token(gqlscan.TokenVarList),
			Token(gqlscan.TokenVarName, "e"),
			Token(gqlscan.TokenVarTypeName, "Episode"),
			Token(gqlscan.TokenVarTypeNotNull),
			Token(gqlscan.TokenEnumVal, "EMPIRE"),
			Token(gqlscan.TokenVarListEnd),
			Token(gqlscan.TokenSet),
			Token(gqlscan.TokenField, "hero"),
			Token(gqlscan.TokenArgList),
			Token(gqlscan.TokenArgName, "episode"),
			gqlparse.MakeVariableIndexToken(0, "e"),
			Token(gqlscan.TokenArgListEnd),
			Token(gqlscan.TokenSet),
			Token(gqlscan.TokenField, "id"),
			Token(gqlscan.TokenSetEnd),
			Token(gqlscan.TokenSetEnd),
		},
	}),
	// Inline variables from JSON
	decl.New(TestSuccess{
		Src:      `query ($v: InputObj! = {foo: "bar"}) {f(a: $v)}`,
		VarsJSON: `{"v": {"foo": "bar from JSON"}}`,
		ExpectVarVals: map[string][]gqlparse.Token{
			"v": { // $v: InputObj!
				Token(gqlscan.TokenObj),
				Token(gqlscan.TokenObjField, "foo"),
				Token(gqlscan.TokenStr, "bar from JSON"),
				Token(gqlscan.TokenObjEnd),
			},
		},
		ExpectOpr: []gqlparse.Token{
			Token(gqlscan.TokenDefQry),
			Token(gqlscan.TokenVarList),

			// $v
			Token(gqlscan.TokenVarName, "v"),
			Token(gqlscan.TokenVarTypeName, "InputObj"),
			Token(gqlscan.TokenVarTypeNotNull),
			Token(gqlscan.TokenObj),
			Token(gqlscan.TokenObjField, "foo"),
			Token(gqlscan.TokenStr, "bar from JSON"),
			Token(gqlscan.TokenObjEnd),

			Token(gqlscan.TokenVarListEnd),
			Token(gqlscan.TokenSet),
			Token(gqlscan.TokenField, "f"),
			Token(gqlscan.TokenArgList),

			// $v
			Token(gqlscan.TokenArgName, "a"),
			gqlparse.MakeVariableIndexToken(0, "v"),

			Token(gqlscan.TokenArgListEnd),
			Token(gqlscan.TokenSetEnd),
		},
	}),
	decl.New(TestSuccess{
		Src:           `{x}`,
		ExpectVarVals: map[string][]gqlparse.Token{},
		ExpectOpr: []gqlparse.Token{
			Token(gqlscan.TokenDefQry),
			Token(gqlscan.TokenSet),
			Token(gqlscan.TokenField, "x"),
			Token(gqlscan.TokenSetEnd),
		},
	}),
	// Inline anonymous fragments
	decl.New(TestSuccess{
		Src:           `{...{x}}`,
		ExpectVarVals: map[string][]gqlparse.Token{},
		ExpectOpr: []gqlparse.Token{
			Token(gqlscan.TokenDefQry),
			Token(gqlscan.TokenSet),
			Token(gqlscan.TokenField, "x"),
			Token(gqlscan.TokenSetEnd),
		},
	}),
	decl.New(TestSuccess{
		Src:           `{...{x ...{y}}}`,
		ExpectVarVals: map[string][]gqlparse.Token{},
		ExpectOpr: []gqlparse.Token{
			Token(gqlscan.TokenDefQry),
			Token(gqlscan.TokenSet),
			Token(gqlscan.TokenField, "x"),
			Token(gqlscan.TokenField, "y"),
			Token(gqlscan.TokenSetEnd),
		},
	}),
	// Inline named fragments
	decl.New(TestSuccess{
		Src:           `{...f}, fragment f on Query {x}`,
		ExpectVarVals: map[string][]gqlparse.Token{},
		ExpectOpr: []gqlparse.Token{
			Token(gqlscan.TokenDefQry),
			Token(gqlscan.TokenSet),
			Token(gqlscan.TokenField, "x"),
			Token(gqlscan.TokenSetEnd),
		},
	}),
	// Inline nested named fragments
	decl.New(TestSuccess{
		Src: `
			fragment a2 on Query { x2, ...b }
			fragment d on Query { ...a2 }
			{ ...a1, ...a2, ...d, s { ...a1 } }
			fragment a1 on Query { ...b }
			fragment b on Query { ...c }
			fragment c on Query { x, y }
		`,
		ExpectVarVals: map[string][]gqlparse.Token{},
		ExpectOpr: []gqlparse.Token{
			Token(gqlscan.TokenDefQry),
			Token(gqlscan.TokenSet),
			Token(gqlscan.TokenField, "x"),
			Token(gqlscan.TokenField, "y"),
			Token(gqlscan.TokenField, "x2"),
			Token(gqlscan.TokenField, "x"),
			Token(gqlscan.TokenField, "y"),
			Token(gqlscan.TokenField, "x2"),
			Token(gqlscan.TokenField, "x"),
			Token(gqlscan.TokenField, "y"),
			Token(gqlscan.TokenField, "s"),
			Token(gqlscan.TokenSet),
			Token(gqlscan.TokenField, "x"),
			Token(gqlscan.TokenField, "y"),
			Token(gqlscan.TokenSetEnd),
			Token(gqlscan.TokenSetEnd),
		},
	}),
	// Inline variables inside fragments
	decl.New(TestSuccess{
		Src: `query X ($v: String! = "text") { ...f1 }
		fragment f1 on Query { foo(bar: $v) }`,
		OprName: "X",
		ExpectVarVals: map[string][]gqlparse.Token{
			"v": {Token(gqlscan.TokenStr, "text")},
		},
		ExpectOpr: []gqlparse.Token{
			Token(gqlscan.TokenDefQry),
			Token(gqlscan.TokenOprName, "X"),
			Token(gqlscan.TokenVarList),

			// $v
			Token(gqlscan.TokenVarName, "v"),
			Token(gqlscan.TokenVarTypeName, "String"),
			Token(gqlscan.TokenVarTypeNotNull),
			Token(gqlscan.TokenStr, "text"),

			Token(gqlscan.TokenVarListEnd),
			Token(gqlscan.TokenSet),

			Token(gqlscan.TokenField, "foo"),
			Token(gqlscan.TokenArgList),
			Token(gqlscan.TokenArgName, "bar"),
			gqlparse.MakeVariableIndexToken(0, "v"),
			Token(gqlscan.TokenArgListEnd),

			Token(gqlscan.TokenSetEnd),
		},
	}),
	decl.New(TestSuccess{
		Src: `query X ($v: String! = "text") {...f1,...f2,...f3}
		fragment f1 on Query { foo(bar: $v), ...f2 }
		fragment f2 on Query { bar(baz: $v) }
		fragment f3 on Query { baz(fuz: $v) }`,
		OprName: "X",
		ExpectVarVals: map[string][]gqlparse.Token{
			"v": {Token(gqlscan.TokenStr, "text")},
		},
		ExpectOpr: []gqlparse.Token{
			Token(gqlscan.TokenDefQry),
			Token(gqlscan.TokenOprName, "X"),
			Token(gqlscan.TokenVarList),

			// $v
			Token(gqlscan.TokenVarName, "v"),
			Token(gqlscan.TokenVarTypeName, "String"),
			Token(gqlscan.TokenVarTypeNotNull),
			Token(gqlscan.TokenStr, "text"),

			Token(gqlscan.TokenVarListEnd),
			Token(gqlscan.TokenSet),

			Token(gqlscan.TokenField, "foo"),
			Token(gqlscan.TokenArgList),
			Token(gqlscan.TokenArgName, "bar"),
			gqlparse.MakeVariableIndexToken(0, "v"),
			Token(gqlscan.TokenArgListEnd),

			Token(gqlscan.TokenField, "bar"),
			Token(gqlscan.TokenArgList),
			Token(gqlscan.TokenArgName, "baz"),
			gqlparse.MakeVariableIndexToken(0, "v"),
			Token(gqlscan.TokenArgListEnd),

			Token(gqlscan.TokenField, "bar"),
			Token(gqlscan.TokenArgList),
			Token(gqlscan.TokenArgName, "baz"),
			gqlparse.MakeVariableIndexToken(0, "v"),
			Token(gqlscan.TokenArgListEnd),

			Token(gqlscan.TokenField, "baz"),
			Token(gqlscan.TokenArgList),
			Token(gqlscan.TokenArgName, "fuz"),
			gqlparse.MakeVariableIndexToken(0, "v"),
			Token(gqlscan.TokenArgListEnd),

			Token(gqlscan.TokenSetEnd),
		},
	}),
	// Inline variables from JSON
	decl.New(TestSuccess{
		OprName: "Q",
		Src: `query Q (
			$v_s: String! = """default value""",
			$v_i: Int! = 42,
			$v_f: Float! = 3.14,
			$v_b: Boolean! = true,
			$v_d: ID! = "default ID",
			$v_o: InputObj! = {foo: "bar"},
			$v_so: String = "default not null",
			$v_io: Int = 42,
			$v_fo: Float = 42.24,
			$v_bo: Boolean = false,
			$v_do: ID = "default not null ID",
			$v_oo: InputObj = {foo: "bar"},
			$v_aon: [String] = ["not null"],
			$v_aoy: [String] = null,
			$v_a_so: [String]! = [],
			$v_a_ao_so: [[String]]! = [],
			$v_a_io: [InputObj]! = [],
		) {
			... on Query {
				f(
					a1: $v_s,
					a2: $v_i,
					a3: $v_f,
					a4: $v_b,
					a5: $v_d,
					a6: $v_o,
					a7: $v_so,
					a8: $v_io,
					a9: $v_fo,
					a10: $v_bo,
					a11: $v_do,
					a12: $v_oo,
					a13: $v_aon,
					a14: $v_aoy,
					a15: $v_a_so,
					a16: $v_a_ao_so,
					a17: $v_a_io,
				)
			}
		}`,
		VarsJSON: `{
			"v_s": "from JSON",
			"v_i": 10042,
			"v_f": 100.314,
			"v_b": false,
			"v_d": "ID from JSON",
			"v_o": {"foo": "bar from JSON"},
			"v_so": null,
			"v_io": null,
			"v_fo": null,
			"v_bo": null,
			"v_do": null,
			"v_oo": null,
			"v_aon": null,
			"v_aoy": [],
			"v_a_so": ["okay", null],
			"v_a_ao_so": [["okay", null], [], null],
			"v_a_io": [{"a": "1", "b": null, "c": 42, "d": false}, null]
		}`,
		ExpectVarVals: map[string][]gqlparse.Token{
			// $v_s: String!
			"v_s": {Token(gqlscan.TokenStr, "from JSON")},

			// $v_i: Int!
			"v_i": {Token(gqlscan.TokenInt, "10042")},

			// $v_f: Float!
			"v_f": {Token(gqlscan.TokenFloat, "100.314")},

			// $v_b: Boolean!
			"v_b": {Token(gqlscan.TokenFalse)},

			// $v_d: ID!
			"v_d": {Token(gqlscan.TokenStr, "ID from JSON")},

			"v_o": { // $v_o: InputObj!
				// {"foo": "bar from JSON"}
				Token(gqlscan.TokenObj),
				Token(gqlscan.TokenObjField, "foo"),
				Token(gqlscan.TokenStr, "bar from JSON"),
				Token(gqlscan.TokenObjEnd),
			},

			// $v_so: String
			"v_so": {Token(gqlscan.TokenNull)},

			// $v_io: Int
			"v_io": {Token(gqlscan.TokenNull)},

			// $v_fo: Float
			"v_fo": {Token(gqlscan.TokenNull)},

			// $v_bo: Boolean
			"v_bo": {Token(gqlscan.TokenNull)},

			// $v_do: ID
			"v_do": {Token(gqlscan.TokenNull)},

			// $v_oo: InputObj
			"v_oo": {Token(gqlscan.TokenNull)},

			// $v_aon: [String]
			"v_aon": {Token(gqlscan.TokenNull)},

			"v_aoy": { // $v_aoy: [String]
				// []
				Token(gqlscan.TokenArr),
				Token(gqlscan.TokenArrEnd),
			},

			"v_a_so": { // $v_a_so: [String]!
				// ["okay", null]
				Token(gqlscan.TokenArr),
				Token(gqlscan.TokenStr, "okay"),
				Token(gqlscan.TokenNull),
				Token(gqlscan.TokenArrEnd),
			},
			"v_a_ao_so": { // $v_a_ao_so: [[String]]!
				// [["okay", null], [], null]
				Token(gqlscan.TokenArr),
				Token(gqlscan.TokenArr),
				Token(gqlscan.TokenStr, "okay"),
				Token(gqlscan.TokenNull),
				Token(gqlscan.TokenArrEnd),
				Token(gqlscan.TokenArr),
				Token(gqlscan.TokenArrEnd),
				Token(gqlscan.TokenNull),
				Token(gqlscan.TokenArrEnd),
			},
			"v_a_io": { // $v_a_io: [InputObj]!
				// [{"a": "1", "b": null, "c": 42, "d": false}, null]
				Token(gqlscan.TokenArr),
				Token(gqlscan.TokenObj),
				Token(gqlscan.TokenObjField, "a"),
				Token(gqlscan.TokenStr, "1"),
				Token(gqlscan.TokenObjField, "b"),
				Token(gqlscan.TokenNull),
				Token(gqlscan.TokenObjField, "c"),
				Token(gqlscan.TokenInt, "42"),
				Token(gqlscan.TokenObjField, "d"),
				Token(gqlscan.TokenFalse),
				Token(gqlscan.TokenObjEnd),
				Token(gqlscan.TokenNull),
				Token(gqlscan.TokenArrEnd),
			},
		},
		ExpectOpr: []gqlparse.Token{
			Token(gqlscan.TokenDefQry),
			Token(gqlscan.TokenOprName, "Q"),
			Token(gqlscan.TokenVarList),

			// $v_s
			Token(gqlscan.TokenVarName, "v_s"),
			Token(gqlscan.TokenVarTypeName, "String"),
			Token(gqlscan.TokenVarTypeNotNull),
			Token(gqlscan.TokenStr, "from JSON"),

			// $v_i
			Token(gqlscan.TokenVarName, "v_i"),
			Token(gqlscan.TokenVarTypeName, "Int"),
			Token(gqlscan.TokenVarTypeNotNull),
			Token(gqlscan.TokenInt, "10042"),

			// $v_f
			Token(gqlscan.TokenVarName, "v_f"),
			Token(gqlscan.TokenVarTypeName, "Float"),
			Token(gqlscan.TokenVarTypeNotNull),
			Token(gqlscan.TokenFloat, "100.314"),

			// $v_b
			Token(gqlscan.TokenVarName, "v_b"),
			Token(gqlscan.TokenVarTypeName, "Boolean"),
			Token(gqlscan.TokenVarTypeNotNull),
			Token(gqlscan.TokenFalse),

			// $v_d
			Token(gqlscan.TokenVarName, "v_d"),
			Token(gqlscan.TokenVarTypeName, "ID"),
			Token(gqlscan.TokenVarTypeNotNull),
			Token(gqlscan.TokenStr, "ID from JSON"),

			// $v_o
			Token(gqlscan.TokenVarName, "v_o"),
			Token(gqlscan.TokenVarTypeName, "InputObj"),
			Token(gqlscan.TokenVarTypeNotNull),
			Token(gqlscan.TokenObj),
			Token(gqlscan.TokenObjField, "foo"),
			Token(gqlscan.TokenStr, "bar from JSON"),
			Token(gqlscan.TokenObjEnd),

			// $v_so
			Token(gqlscan.TokenVarName, "v_so"),
			Token(gqlscan.TokenVarTypeName, "String"),
			Token(gqlscan.TokenNull),

			// $v_io
			Token(gqlscan.TokenVarName, "v_io"),
			Token(gqlscan.TokenVarTypeName, "Int"),
			Token(gqlscan.TokenNull),

			// $v_fo
			Token(gqlscan.TokenVarName, "v_fo"),
			Token(gqlscan.TokenVarTypeName, "Float"),
			Token(gqlscan.TokenNull),

			// $v_bo
			Token(gqlscan.TokenVarName, "v_bo"),
			Token(gqlscan.TokenVarTypeName, "Boolean"),
			Token(gqlscan.TokenNull),

			// $v_do
			Token(gqlscan.TokenVarName, "v_do"),
			Token(gqlscan.TokenVarTypeName, "ID"),
			Token(gqlscan.TokenNull),

			// $v_oo
			Token(gqlscan.TokenVarName, "v_oo"),
			Token(gqlscan.TokenVarTypeName, "InputObj"),
			Token(gqlscan.TokenNull),

			// $v_aon
			Token(gqlscan.TokenVarName, "v_aon"),
			Token(gqlscan.TokenVarTypeArr),
			Token(gqlscan.TokenVarTypeName, "String"),
			Token(gqlscan.TokenVarTypeArrEnd),
			Token(gqlscan.TokenNull),

			// $v_aoy
			Token(gqlscan.TokenVarName, "v_aoy"),
			Token(gqlscan.TokenVarTypeArr),
			Token(gqlscan.TokenVarTypeName, "String"),
			Token(gqlscan.TokenVarTypeArrEnd),
			Token(gqlscan.TokenArr),
			Token(gqlscan.TokenArrEnd),

			// $v_a_so
			Token(gqlscan.TokenVarName, "v_a_so"),
			Token(gqlscan.TokenVarTypeArr),
			Token(gqlscan.TokenVarTypeName, "String"),
			Token(gqlscan.TokenVarTypeArrEnd),
			Token(gqlscan.TokenVarTypeNotNull),
			Token(gqlscan.TokenArr),
			Token(gqlscan.TokenStr, "okay"),
			Token(gqlscan.TokenNull),
			Token(gqlscan.TokenArrEnd),

			// $v_a_ao_so
			Token(gqlscan.TokenVarName, "v_a_ao_so"),
			Token(gqlscan.TokenVarTypeArr),
			Token(gqlscan.TokenVarTypeArr),
			Token(gqlscan.TokenVarTypeName, "String"),
			Token(gqlscan.TokenVarTypeArrEnd),
			Token(gqlscan.TokenVarTypeArrEnd),
			Token(gqlscan.TokenVarTypeNotNull),
			Token(gqlscan.TokenArr),
			Token(gqlscan.TokenArr),
			Token(gqlscan.TokenStr, "okay"),
			Token(gqlscan.TokenNull),
			Token(gqlscan.TokenArrEnd),
			Token(gqlscan.TokenArr),
			Token(gqlscan.TokenArrEnd),
			Token(gqlscan.TokenNull),
			Token(gqlscan.TokenArrEnd),

			// $v_a_io
			Token(gqlscan.TokenVarName, "v_a_io"),
			Token(gqlscan.TokenVarTypeArr),
			Token(gqlscan.TokenVarTypeName, "InputObj"),
			Token(gqlscan.TokenVarTypeArrEnd),
			Token(gqlscan.TokenVarTypeNotNull),
			Token(gqlscan.TokenArr),
			Token(gqlscan.TokenObj),
			Token(gqlscan.TokenObjField, "a"),
			Token(gqlscan.TokenStr, "1"),
			Token(gqlscan.TokenObjField, "b"),
			Token(gqlscan.TokenNull),
			Token(gqlscan.TokenObjField, "c"),
			Token(gqlscan.TokenInt, "42"),
			Token(gqlscan.TokenObjField, "d"),
			Token(gqlscan.TokenFalse),
			Token(gqlscan.TokenObjEnd),
			Token(gqlscan.TokenNull),
			Token(gqlscan.TokenArrEnd),

			Token(gqlscan.TokenVarListEnd),
			Token(gqlscan.TokenSet),
			Token(gqlscan.TokenFragInline, "Query"),
			Token(gqlscan.TokenSet),
			Token(gqlscan.TokenField, "f"),
			Token(gqlscan.TokenArgList),

			// $v_s: String! = """default value""",
			Token(gqlscan.TokenArgName, "a1"),
			gqlparse.MakeVariableIndexToken(0, "v_s"),

			// $v_i: Int! = 42,
			Token(gqlscan.TokenArgName, "a2"),
			gqlparse.MakeVariableIndexToken(1, "v_i"),

			// $v_f: Float! = 3.14,
			Token(gqlscan.TokenArgName, "a3"),
			gqlparse.MakeVariableIndexToken(2, "v_f"),

			// $v_b: Boolean! = true,
			Token(gqlscan.TokenArgName, "a4"),
			gqlparse.MakeVariableIndexToken(3, "v_b"),

			// $v_d: ID! = "default ID",
			Token(gqlscan.TokenArgName, "a5"),
			gqlparse.MakeVariableIndexToken(4, "v_d"),

			// $v_o: InputObj! = {foo: "bar"},
			Token(gqlscan.TokenArgName, "a6"),
			gqlparse.MakeVariableIndexToken(5, "v_o"),

			// $v_so: String = null,
			Token(gqlscan.TokenArgName, "a7"),
			gqlparse.MakeVariableIndexToken(6, "v_so"),

			// $v_io: Int = null,
			Token(gqlscan.TokenArgName, "a8"),
			gqlparse.MakeVariableIndexToken(7, "v_io"),

			// $v_fo: Float = null,
			Token(gqlscan.TokenArgName, "a9"),
			gqlparse.MakeVariableIndexToken(8, "v_fo"),

			// $v_bo: Boolean = null,
			Token(gqlscan.TokenArgName, "a10"),
			gqlparse.MakeVariableIndexToken(9, "v_bo"),

			// $v_do: ID = null,
			Token(gqlscan.TokenArgName, "a11"),
			gqlparse.MakeVariableIndexToken(10, "v_do"),

			// $v_oo: InputObj = null,
			Token(gqlscan.TokenArgName, "a12"),
			gqlparse.MakeVariableIndexToken(11, "v_oo"),

			// $v_aon: [String] = null,
			Token(gqlscan.TokenArgName, "a13"),
			gqlparse.MakeVariableIndexToken(12, "v_aon"),

			// $v_aoy: [String] = [],
			Token(gqlscan.TokenArgName, "a14"),
			gqlparse.MakeVariableIndexToken(13, "v_aoy"),

			// $v_a_so: [String]! = ["okay", null],
			Token(gqlscan.TokenArgName, "a15"),
			gqlparse.MakeVariableIndexToken(14, "v_a_so"),

			// $v_a_ao_so: [[String]]! = [["okay", null], [], null],
			Token(gqlscan.TokenArgName, "a16"),
			gqlparse.MakeVariableIndexToken(15, "v_a_ao_so"),

			// $v_a_io: [InputObj]! = [{a: "1", b: null, c: 42, d: false}, null],
			Token(gqlscan.TokenArgName, "a17"),
			gqlparse.MakeVariableIndexToken(16, "v_a_io"),

			Token(gqlscan.TokenArgListEnd),
			Token(gqlscan.TokenSetEnd),
			Token(gqlscan.TokenSetEnd),
		},
	}),
	// Inline variables with default values
	decl.New(TestSuccess{
		OprName: "Q",
		Src: `query Q (
			$v_s: String! = """default value""",
			$v_i: Int! = 42,
			$v_f: Float! = 3.14,
			$v_b: Boolean! = true,
			$v_d: ID! = "default ID",
			$v_o: InputObj! = {foo: "bar"},
			$v_so: String = null,
			$v_io: Int = null,
			$v_fo: Float = null,
			$v_bo: Boolean = null,
			$v_do: ID = null,
			$v_oo: InputObj = null,
			$v_aon: [String] = null,
			$v_aoy: [String] = [],
			$v_a_so: [String]! = ["okay", null],
			$v_a_ao_so: [[String]]! = [["okay", null], [], null],
			$v_a_io: [InputObj]! = [{a: "1", b: null, c: 42, d: false}, null],
		) {
			... on Query {
				f(
					a1: $v_s,
					a2: $v_i,
					a3: $v_f,
					a4: $v_b,
					a5: $v_d,
					a6: $v_o,
					a7: $v_so,
					a8: $v_io,
					a9: $v_fo,
					a10: $v_bo,
					a11: $v_do,
					a12: $v_oo,
					a13: $v_aon,
					a14: $v_aoy,
					a15: $v_a_so,
					a16: $v_a_ao_so,
					a17: $v_a_io,
				)
			}
		}`,
		ExpectVarVals: map[string][]gqlparse.Token{
			"v_s": {Token(gqlscan.TokenStrBlock, "default value")},
			"v_i": {Token(gqlscan.TokenInt, "42")},
			"v_f": {Token(gqlscan.TokenFloat, "3.14")},
			"v_b": {Token(gqlscan.TokenTrue)},
			"v_d": {Token(gqlscan.TokenStr, "default ID")},
			"v_o": {
				Token(gqlscan.TokenObj),
				Token(gqlscan.TokenObjField, "foo"),
				Token(gqlscan.TokenStr, "bar"),
				Token(gqlscan.TokenObjEnd),
			},
			"v_so":  {Token(gqlscan.TokenNull)},
			"v_io":  {Token(gqlscan.TokenNull)},
			"v_fo":  {Token(gqlscan.TokenNull)},
			"v_bo":  {Token(gqlscan.TokenNull)},
			"v_do":  {Token(gqlscan.TokenNull)},
			"v_oo":  {Token(gqlscan.TokenNull)},
			"v_aon": {Token(gqlscan.TokenNull)},
			"v_aoy": {
				Token(gqlscan.TokenArr),
				Token(gqlscan.TokenArrEnd),
			},
			"v_a_so": {
				Token(gqlscan.TokenArr),
				Token(gqlscan.TokenStr, "okay"),
				Token(gqlscan.TokenNull),
				Token(gqlscan.TokenArrEnd),
			},
			"v_a_ao_so": {
				Token(gqlscan.TokenArr),
				Token(gqlscan.TokenArr),
				Token(gqlscan.TokenStr, "okay"),
				Token(gqlscan.TokenNull),
				Token(gqlscan.TokenArrEnd),
				Token(gqlscan.TokenArr),
				Token(gqlscan.TokenArrEnd),
				Token(gqlscan.TokenNull),
				Token(gqlscan.TokenArrEnd),
			},
			"v_a_io": {
				Token(gqlscan.TokenArr),
				Token(gqlscan.TokenObj),
				Token(gqlscan.TokenObjField, "a"),
				Token(gqlscan.TokenStr, "1"),
				Token(gqlscan.TokenObjField, "b"),
				Token(gqlscan.TokenNull),
				Token(gqlscan.TokenObjField, "c"),
				Token(gqlscan.TokenInt, "42"),
				Token(gqlscan.TokenObjField, "d"),
				Token(gqlscan.TokenFalse),
				Token(gqlscan.TokenObjEnd),
				Token(gqlscan.TokenNull),
				Token(gqlscan.TokenArrEnd),
			},
		},
		ExpectOpr: []gqlparse.Token{
			Token(gqlscan.TokenDefQry),
			Token(gqlscan.TokenOprName, "Q"),
			Token(gqlscan.TokenVarList),

			// $v_s: String! = """default value"""
			Token(gqlscan.TokenVarName, "v_s"),
			Token(gqlscan.TokenVarTypeName, "String"),
			Token(gqlscan.TokenVarTypeNotNull),
			Token(gqlscan.TokenStrBlock, "default value"),

			// $v_i: Int! = 42
			Token(gqlscan.TokenVarName, "v_i"),
			Token(gqlscan.TokenVarTypeName, "Int"),
			Token(gqlscan.TokenVarTypeNotNull),
			Token(gqlscan.TokenInt, "42"),

			// $v_f: Float! = 3.14
			Token(gqlscan.TokenVarName, "v_f"),
			Token(gqlscan.TokenVarTypeName, "Float"),
			Token(gqlscan.TokenVarTypeNotNull),
			Token(gqlscan.TokenFloat, "3.14"),

			// $v_b: Boolean! = true
			Token(gqlscan.TokenVarName, "v_b"),
			Token(gqlscan.TokenVarTypeName, "Boolean"),
			Token(gqlscan.TokenVarTypeNotNull),
			Token(gqlscan.TokenTrue),

			// $v_d: ID! = "default ID"
			Token(gqlscan.TokenVarName, "v_d"),
			Token(gqlscan.TokenVarTypeName, "ID"),
			Token(gqlscan.TokenVarTypeNotNull),
			Token(gqlscan.TokenStr, "default ID"),

			// $v_o: InputObj! = {foo: "bar"}
			Token(gqlscan.TokenVarName, "v_o"),
			Token(gqlscan.TokenVarTypeName, "InputObj"),
			Token(gqlscan.TokenVarTypeNotNull),
			Token(gqlscan.TokenObj),
			Token(gqlscan.TokenObjField, "foo"),
			Token(gqlscan.TokenStr, "bar"),
			Token(gqlscan.TokenObjEnd),

			// $v_so: String = null
			Token(gqlscan.TokenVarName, "v_so"),
			Token(gqlscan.TokenVarTypeName, "String"),
			Token(gqlscan.TokenNull),

			// $v_io: Int = null
			Token(gqlscan.TokenVarName, "v_io"),
			Token(gqlscan.TokenVarTypeName, "Int"),
			Token(gqlscan.TokenNull),

			// $v_fo: Float = null
			Token(gqlscan.TokenVarName, "v_fo"),
			Token(gqlscan.TokenVarTypeName, "Float"),
			Token(gqlscan.TokenNull),

			// $v_bo: Boolean = null
			Token(gqlscan.TokenVarName, "v_bo"),
			Token(gqlscan.TokenVarTypeName, "Boolean"),
			Token(gqlscan.TokenNull),

			// $v_do: ID = null
			Token(gqlscan.TokenVarName, "v_do"),
			Token(gqlscan.TokenVarTypeName, "ID"),
			Token(gqlscan.TokenNull),

			// $v_oo: InputObj = null
			Token(gqlscan.TokenVarName, "v_oo"),
			Token(gqlscan.TokenVarTypeName, "InputObj"),
			Token(gqlscan.TokenNull),

			// $v_aon: [String] = null
			Token(gqlscan.TokenVarName, "v_aon"),
			Token(gqlscan.TokenVarTypeArr),
			Token(gqlscan.TokenVarTypeName, "String"),
			Token(gqlscan.TokenVarTypeArrEnd),
			Token(gqlscan.TokenNull),

			// $v_aoy: [String] = []
			Token(gqlscan.TokenVarName, "v_aoy"),
			Token(gqlscan.TokenVarTypeArr),
			Token(gqlscan.TokenVarTypeName, "String"),
			Token(gqlscan.TokenVarTypeArrEnd),
			Token(gqlscan.TokenArr),
			Token(gqlscan.TokenArrEnd),

			// $v_a_so: [String]! = ["okay", null]
			Token(gqlscan.TokenVarName, "v_a_so"),
			Token(gqlscan.TokenVarTypeArr),
			Token(gqlscan.TokenVarTypeName, "String"),
			Token(gqlscan.TokenVarTypeArrEnd),
			Token(gqlscan.TokenVarTypeNotNull),
			Token(gqlscan.TokenArr),
			Token(gqlscan.TokenStr, "okay"),
			Token(gqlscan.TokenNull),
			Token(gqlscan.TokenArrEnd),

			// $v_a_ao_so: [[String]]! = [["okay", null], [], null]
			Token(gqlscan.TokenVarName, "v_a_ao_so"),
			Token(gqlscan.TokenVarTypeArr),
			Token(gqlscan.TokenVarTypeArr),
			Token(gqlscan.TokenVarTypeName, "String"),
			Token(gqlscan.TokenVarTypeArrEnd),
			Token(gqlscan.TokenVarTypeArrEnd),
			Token(gqlscan.TokenVarTypeNotNull),
			Token(gqlscan.TokenArr),
			Token(gqlscan.TokenArr),
			Token(gqlscan.TokenStr, "okay"),
			Token(gqlscan.TokenNull),
			Token(gqlscan.TokenArrEnd),
			Token(gqlscan.TokenArr),
			Token(gqlscan.TokenArrEnd),
			Token(gqlscan.TokenNull),
			Token(gqlscan.TokenArrEnd),

			// $v_a_io: [InputObj]! = [{a: "1", b: null, c: 42, d: false}, null]
			Token(gqlscan.TokenVarName, "v_a_io"),
			Token(gqlscan.TokenVarTypeArr),
			Token(gqlscan.TokenVarTypeName, "InputObj"),
			Token(gqlscan.TokenVarTypeArrEnd),
			Token(gqlscan.TokenVarTypeNotNull),
			Token(gqlscan.TokenArr),
			Token(gqlscan.TokenObj),
			Token(gqlscan.TokenObjField, "a"),
			Token(gqlscan.TokenStr, "1"),
			Token(gqlscan.TokenObjField, "b"),
			Token(gqlscan.TokenNull),
			Token(gqlscan.TokenObjField, "c"),
			Token(gqlscan.TokenInt, "42"),
			Token(gqlscan.TokenObjField, "d"),
			Token(gqlscan.TokenFalse),
			Token(gqlscan.TokenObjEnd),
			Token(gqlscan.TokenNull),
			Token(gqlscan.TokenArrEnd),

			Token(gqlscan.TokenVarListEnd),
			Token(gqlscan.TokenSet),
			Token(gqlscan.TokenFragInline, "Query"),
			Token(gqlscan.TokenSet),
			Token(gqlscan.TokenField, "f"),
			Token(gqlscan.TokenArgList),

			// $v_s: String! = """default value""",
			Token(gqlscan.TokenArgName, "a1"),
			gqlparse.MakeVariableIndexToken(0, "v_s"),

			// $v_i: Int! = 42,
			Token(gqlscan.TokenArgName, "a2"),
			gqlparse.MakeVariableIndexToken(1, "v_i"),

			// $v_f: Float! = 3.14,
			Token(gqlscan.TokenArgName, "a3"),
			gqlparse.MakeVariableIndexToken(2, "v_f"),

			// $v_b: Boolean! = true,
			Token(gqlscan.TokenArgName, "a4"),
			gqlparse.MakeVariableIndexToken(3, "v_b"),

			// $v_d: ID! = "default ID",
			Token(gqlscan.TokenArgName, "a5"),
			gqlparse.MakeVariableIndexToken(4, "v_d"),

			// $v_o: InputObj! = {foo: "bar"},
			Token(gqlscan.TokenArgName, "a6"),
			gqlparse.MakeVariableIndexToken(5, "v_o"),

			// $v_so: String = null,
			Token(gqlscan.TokenArgName, "a7"),
			gqlparse.MakeVariableIndexToken(6, "v_so"),

			// $v_io: Int = null,
			Token(gqlscan.TokenArgName, "a8"),
			gqlparse.MakeVariableIndexToken(7, "v_io"),

			// $v_fo: Float = null,
			Token(gqlscan.TokenArgName, "a9"),
			gqlparse.MakeVariableIndexToken(8, "v_fo"),

			// $v_bo: Boolean = null,
			Token(gqlscan.TokenArgName, "a10"),
			gqlparse.MakeVariableIndexToken(9, "v_bo"),

			// $v_do: ID = null,
			Token(gqlscan.TokenArgName, "a11"),
			gqlparse.MakeVariableIndexToken(10, "v_do"),

			// $v_oo: InputObj = null,
			Token(gqlscan.TokenArgName, "a12"),
			gqlparse.MakeVariableIndexToken(11, "v_oo"),

			// $v_aon: [String] = null,
			Token(gqlscan.TokenArgName, "a13"),
			gqlparse.MakeVariableIndexToken(12, "v_aon"),

			// $v_aoy: [String] = [],
			Token(gqlscan.TokenArgName, "a14"),
			gqlparse.MakeVariableIndexToken(13, "v_aoy"),

			// $v_a_so: [String]! = ["okay", null],
			Token(gqlscan.TokenArgName, "a15"),
			gqlparse.MakeVariableIndexToken(14, "v_a_so"),

			// $v_a_ao_so: [[String]]! = [["okay", null], [], null],
			Token(gqlscan.TokenArgName, "a16"),
			gqlparse.MakeVariableIndexToken(15, "v_a_ao_so"),

			// $v_a_io: [InputObj]! = [{a: "1", b: null, c: 42, d: false}, null],
			Token(gqlscan.TokenArgName, "a17"),
			gqlparse.MakeVariableIndexToken(16, "v_a_io"),

			Token(gqlscan.TokenArgListEnd),
			Token(gqlscan.TokenSetEnd),
			Token(gqlscan.TokenSetEnd),
		},
	}),
}

func TestOK(t *testing.T) {
	for _, td := range testdata {
		t.Run(td.Decl, func(t *testing.T) {
			var expectOriginal []gqlparse.Token
			err := gqlscan.ScanAll([]byte(td.Data.Src), func(i *gqlscan.Iterator) {
				expectOriginal = append(expectOriginal, gqlparse.Token{
					ID:    i.Token(),
					Value: i.Value(),
				})
			})
			require.False(t, err.IsErr())

			gqlparse.NewParser().Parse(
				[]byte(td.Data.Src),
				[]byte(td.Data.OprName),
				[]byte(td.Data.VarsJSON),
				func(
					varValue [][]gqlparse.Token,
					operation []gqlparse.Token,
					selectionSet []gqlparse.Token,
				) {
					// fmt.Printf("expected: (%d)\n", len(td.Data.ExpectOpr))
					// for i, x := range td.Data.ExpectOpr {
					// 	fmt.Printf(" %d: ", i)
					// 	if i := x.VariableIndex(); i > -1 {
					// 		fmt.Printf("variable value identifier (%d)", i)
					// 	} else {
					// 		fmt.Printf(" %v", x.ID)
					// 	}
					// 	if x.Value == nil {
					// 		fmt.Print("\n")
					// 	} else {
					// 		fmt.Printf(" (%q)\n", string(x.Value))
					// 	}
					// }
					// fmt.Println(" ")
					// fmt.Printf("operation: (%d)\n", len(operation))
					// for i, x := range operation {
					// 	fmt.Printf(" %d: ", i)
					// 	if i := x.VariableIndex(); i > -1 {
					// 		fmt.Printf("variable value identifier (%d)", i)
					// 	} else {
					// 		fmt.Printf(" %v", x.ID)
					// 	}
					// 	if x.Value == nil {
					// 		fmt.Print("\n")
					// 	} else {
					// 		fmt.Printf(" (%q)\n", string(x.Value))
					// 	}
					// }
					// fmt.Println(" ")

					testeq.Slices(
						t, "token",
						td.Data.ExpectOpr, operation,
						func(expected, actual gqlparse.Token) (errMsg string) {
							if expected.ID != actual.ID ||
								string(expected.Value) != string(actual.Value) {
								return fmt.Sprintf(
									"expected {%s}; received: {%s}",
									stringifyToken(expected),
									stringifyToken(actual),
								)
							}
							return ""
						},
						stringifyToken,
					)

					variableValues := make(map[string][]gqlparse.Token)
					for _, t := range operation {
						if i := t.VariableIndex(); i > -1 {
							variableValues[string(t.Value)] = varValue[i]
						}
					}

					testeq.Maps(
						t, "variable value",
						td.Data.ExpectVarVals, variableValues,
						func(expected, actual []gqlparse.Token) (errMsg string) {
							if !testeq.Slices(
								t, "token", expected, actual,
								func(expected, actual gqlparse.Token) (errMsg string) {
									if expected.ID != actual.ID ||
										string(expected.Value) != string(actual.Value) {
										return fmt.Sprintf(
											"expected {%s}; received: {%s}",
											stringifyToken(expected),
											stringifyToken(actual),
										)
									}
									return ""
								},
								func(t gqlparse.Token) string {
									return stringifyToken(t)
								},
							) {
								return fmt.Sprintf(
									"expected: %v; received: %v",
									expected, actual,
								)
							}
							return ""
						},
						func(value []gqlparse.Token) string {
							return fmt.Sprintf("%v", value)
						},
					)

					require.Equal(t,
						operation[len(operation)-len(selectionSet):],
						selectionSet,
					)
					require.Equal(t,
						gqlscan.TokenSet.String(),
						selectionSet[0].ID.String(),
					)
				},
				func(err error) {
					t.Fatal("unexpected error:", err.Error())
				},
			)
		})
	}
}

var testdataErr = []decl.Declaration[TestError]{
	// Operation not found
	decl.New(TestError{
		Src: `query A {x}, query B {x}`,
		Check: func(t *testing.T, err error) {
			require.Equal(t, &gqlparse.ErrorOprNotFound{
				OperationName: []byte(""),
			}, err)
			require.Equal(t, `operation "" not found`, err.Error())
		},
	}),
	decl.New(TestError{
		Src:     `query A {x}, query B {x}`,
		OprName: "C",
		Check: func(t *testing.T, err error) {
			require.Equal(t, &gqlparse.ErrorOprNotFound{
				OperationName: []byte("C"),
			}, err)
			require.Equal(t, `operation "C" not found`, err.Error())
		},
	}),
	decl.New(TestError{
		Src:     `{x}`,
		OprName: "A",
		Check: func(t *testing.T, err error) {
			require.Equal(t, &gqlparse.ErrorOprNotFound{
				OperationName: []byte("A"),
			}, err)
			require.Equal(t, `operation "A" not found`, err.Error())
		},
	}),

	// Non-exclusive anonymous operation
	decl.New(TestError{
		Src: `{x}, query{x}`,
		Check: func(t *testing.T, err error) {
			require.Equal(t, &gqlparse.ErrorOprAnonNonExcl{}, err)
			require.Equal(t, `non-exclusive anonymous operation`, err.Error())
		},
	}),
	decl.New(TestError{
		Src: `{x}, mutation M {x}`,
		Check: func(t *testing.T, err error) {
			require.Equal(t, &gqlparse.ErrorOprAnonNonExcl{}, err)
			require.Equal(t, `non-exclusive anonymous operation`, err.Error())
		},
	}),
	decl.New(TestError{
		Src: `query A {x}, query {x}`,
		Check: func(t *testing.T, err error) {
			require.Equal(t, &gqlparse.ErrorOprAnonNonExcl{}, err)
			require.Equal(t, `non-exclusive anonymous operation`, err.Error())
		},
	}),

	// Redeclared operation
	decl.New(TestError{
		Src: `query A {x}, query A {x}`,
		Check: func(t *testing.T, err error) {
			require.Equal(t, &gqlparse.ErrorRedecOpr{
				OperationName: []byte("A"),
			}, err)
			require.Equal(t, `operation "A" redeclared`, err.Error())
		},
	}),
	decl.New(TestError{
		Src: `query M {x}, mutation M {x}`,
		Check: func(t *testing.T, err error) {
			require.Equal(t, &gqlparse.ErrorRedecOpr{
				OperationName: []byte("M"),
			}, err)
			require.Equal(t, `operation "M" redeclared`, err.Error())
		},
	}),
	decl.New(TestError{
		Src: `query S {x}, subscription S {x}`,
		Check: func(t *testing.T, err error) {
			require.Equal(t, &gqlparse.ErrorRedecOpr{
				OperationName: []byte("S"),
			}, err)
			require.Equal(t, `operation "S" redeclared`, err.Error())
		},
	}),

	// Redeclared fragment
	decl.New(TestError{
		Src: `{...f}
			fragment f on Query {x}
			fragment f on Query {x}`,
		Check: func(t *testing.T, err error) {
			require.Equal(t, &gqlparse.ErrorRedecFrag{
				FragmentName: []byte("f"),
			}, err)
			require.Equal(t, `fragment "f" redeclared`, err.Error())
		},
	}),

	// Unused fragment
	decl.New(TestError{
		Src: `{...f}
			fragment f on Query {x}
			fragment a on Query {x}`,
		Check: func(t *testing.T, err error) {
			require.Equal(t, &gqlparse.ErrorFragUnused{
				FragmentName: []byte("a"),
			}, err)
			require.Equal(t, `fragment "a" unused`, err.Error())
		},
	}),

	// Recursive fragment
	decl.New(TestError{
		Src: `{...a}
			fragment a on Query {...a}`,
		Check: func(t *testing.T, err error) {
			require.Equal(t, &gqlparse.ErrorFragRecurse{
				Path: [][]byte{
					[]byte("a"),
					[]byte("a"),
				},
			}, err)
			require.Equal(
				t, `fragment recursion detected at: a.a`, err.Error(),
			)
		},
	}),
	decl.New(TestError{
		Src: `{...a}
			fragment a on Query {...b}
			fragment b on Query {...a}`,
		Check: func(t *testing.T, err error) {
			require.Equal(t, &gqlparse.ErrorFragRecurse{
				Path: [][]byte{
					[]byte("a"),
					[]byte("b"),
					[]byte("a"),
				},
			}, err)
			require.Equal(
				t, `fragment recursion detected at: a.b.a`, err.Error(),
			)
		},
	}),
	decl.New(TestError{
		Src: `{...a, ...a1}
			fragment a on Query {...b}
			fragment a1 on Query {...b}
			fragment b on Query {...c}
			fragment c on Query {...a1}`,
		Check: func(t *testing.T, err error) {
			require.Equal(t, &gqlparse.ErrorFragRecurse{
				Path: [][]byte{
					[]byte("b"),
					[]byte("c"),
					[]byte("a1"),
					[]byte("b"),
				},
			}, err)
			require.Equal(
				t, `fragment recursion detected at: b.c.a1.b`, err.Error(),
			)
		},
	}),

	// Redeclared variable
	decl.New(TestError{
		Src: `query($v1:String, $v1:Int){f(a:$v1)}`,
		Check: func(t *testing.T, err error) {
			require.Equal(t, &gqlparse.ErrorRedeclVar{
				VariableName: []byte("v1"),
			}, err)
			require.Equal(t, `variable "v1" redeclared`, err.Error())
		},
	}),
	decl.New(TestError{
		Src:     `query Q ($v2:String, $v2:Int){f(a:$v2)}`,
		OprName: "Q",
		Check: func(t *testing.T, err error) {
			require.Equal(t, &gqlparse.ErrorRedeclVar{
				VariableName: []byte("v2"),
			}, err)
			require.Equal(t, `variable "v2" redeclared`, err.Error())
		},
	}),

	// Default value wrong type
	decl.New(TestError{
		Src: `query ($v:String=true) { f(a:$v) }`,
		Check: func(t *testing.T, err error) {
			require.Error(t, err)
			require.IsType(t, &gqlparse.ErrorUnexpValType{}, err)
			require.Equal(
				t,
				"unexpected value type, "+
					"expected: String; "+
					"received(default): true",
				err.Error(),
			)
		},
	}),
	decl.New(TestError{
		Src: `query ($v:[String]="okay") { f(a:$v) }`,
		Check: func(t *testing.T, err error) {
			require.Error(t, err)
			require.IsType(t, &gqlparse.ErrorUnexpValType{}, err)
			require.Equal(
				t,
				"unexpected value type, "+
					"expected: [String]; "+
					`received(default): "okay"`,
				err.Error(),
			)
		},
	}),
	decl.New(TestError{
		Src: `query ($v:Int=42.5) { f(a:$v) }`,
		Check: func(t *testing.T, err error) {
			require.Error(t, err)
			require.IsType(t, &gqlparse.ErrorUnexpValType{}, err)
			require.Equal(
				t,
				"unexpected value type, "+
					"expected: Int; "+
					`received(default): 42.5`,
				err.Error(),
			)
		},
	}),
	decl.New(TestError{
		Src: `query ($v:Float=false) { f(a:$v) }`,
		Check: func(t *testing.T, err error) {
			require.Error(t, err)
			require.IsType(t, &gqlparse.ErrorUnexpValType{}, err)
			require.Equal(
				t,
				"unexpected value type, "+
					"expected: Float; "+
					`received(default): false`,
				err.Error(),
			)
		},
	}),
	decl.New(TestError{
		Src: `query ($v:Boolean=1) { f(a:$v) }`,
		Check: func(t *testing.T, err error) {
			require.Error(t, err)
			require.IsType(t, &gqlparse.ErrorUnexpValType{}, err)
			require.Equal(
				t,
				"unexpected value type, "+
					"expected: Boolean; "+
					`received(default): 1`,
				err.Error(),
			)
		},
	}),
	decl.New(TestError{
		Src: `query ($v:Input=1) { f(a:$v) }`,
		Check: func(t *testing.T, err error) {
			require.Error(t, err)
			require.IsType(t, &gqlparse.ErrorUnexpValType{}, err)
			require.Equal(
				t,
				"unexpected value type, "+
					"expected: Input; "+
					`received(default): 1`,
				err.Error(),
			)
		},
	}),
	decl.New(TestError{
		Src: `query ($v:Int={foo:"bar", baz:42}) { f(a:$v) }`,
		Check: func(t *testing.T, err error) {
			require.Error(t, err)
			require.IsType(t, &gqlparse.ErrorUnexpValType{}, err)
			require.Equal(
				t,
				"unexpected value type, "+
					"expected: Int; "+
					`received(default): {foo:"bar",baz:42}`,
				err.Error(),
			)
		},
	}),
	decl.New(TestError{
		Src: `query ($v:Int=[]) { f(a:$v) }`,
		Check: func(t *testing.T, err error) {
			require.Error(t, err)
			require.IsType(t, &gqlparse.ErrorUnexpValType{}, err)
			require.Equal(
				t,
				"unexpected value type, "+
					"expected: Int; "+
					`received(default): []`,
				err.Error(),
			)
		},
	}),
	decl.New(TestError{
		Src: `query ($v:Int=[{x:2,y:4}, null, {x:8,y:8}]) { f(a:$v) }`,
		Check: func(t *testing.T, err error) {
			require.Error(t, err)
			require.IsType(t, &gqlparse.ErrorUnexpValType{}, err)
			require.Equal(
				t,
				"unexpected value type, "+
					"expected: Int; "+
					`received(default): [{x:2,y:4},null,{x:8,y:8}]`,
				err.Error(),
			)
		},
	}),
	decl.New(TestError{
		Src: `query ($v:Int! = null) { f(a:$v) }`,
		Check: func(t *testing.T, err error) {
			require.Error(t, err)
			require.IsType(t, &gqlparse.ErrorUnexpValType{}, err)
			require.Equal(
				t,
				"unexpected value type, "+
					"expected: Int!; "+
					`received(default): null`,
				err.Error(),
			)
		},
	}),

	// JSON variable wrong type
	decl.New(TestError{
		Src:      `query ($v:String) { f(a:$v) }`,
		VarsJSON: `{"v":true}`,
		Check: func(t *testing.T, err error) {
			require.Error(t, err)
			require.IsType(t, &gqlparse.ErrorUnexpValType{}, err)
			require.Equal(
				t,
				"unexpected value type, "+
					"expected: String; "+
					"received(json): true",
				err.Error(),
			)
		},
	}),
	decl.New(TestError{
		Src:      `query ($v:[String]) { f(a:$v) }`,
		VarsJSON: `{"v":"okay"}`,
		Check: func(t *testing.T, err error) {
			require.Error(t, err)
			require.IsType(t, &gqlparse.ErrorUnexpValType{}, err)
			require.Equal(
				t,
				"unexpected value type, "+
					"expected: [String]; "+
					`received(json): "okay"`,
				err.Error(),
			)
		},
	}),
	decl.New(TestError{
		Src:      `query ($v:Int) { f(a:$v) }`,
		VarsJSON: `{"v":42.5}`,
		Check: func(t *testing.T, err error) {
			require.Error(t, err)
			require.IsType(t, &gqlparse.ErrorUnexpValType{}, err)
			require.Equal(
				t,
				"unexpected value type, "+
					"expected: Int; "+
					`received(json): 42.5`,
				err.Error(),
			)
		},
	}),
	decl.New(TestError{
		Src:      `query ($v:Float) { f(a:$v) }`,
		VarsJSON: `{"v":false}`,
		Check: func(t *testing.T, err error) {
			require.Error(t, err)
			require.IsType(t, &gqlparse.ErrorUnexpValType{}, err)
			require.Equal(
				t,
				"unexpected value type, "+
					"expected: Float; "+
					`received(json): false`,
				err.Error(),
			)
		},
	}),
	decl.New(TestError{
		Src:      `query ($v:Boolean) { f(a:$v) }`,
		VarsJSON: `{"v":1}`,
		Check: func(t *testing.T, err error) {
			require.Error(t, err)
			require.IsType(t, &gqlparse.ErrorUnexpValType{}, err)
			require.Equal(
				t,
				"unexpected value type, "+
					"expected: Boolean; "+
					`received(json): 1`,
				err.Error(),
			)
		},
	}),
	decl.New(TestError{
		Src:      `query ($v:Input) { f(a:$v) }`,
		VarsJSON: `{"v":1}`,
		Check: func(t *testing.T, err error) {
			require.Error(t, err)
			require.IsType(t, &gqlparse.ErrorUnexpValType{}, err)
			require.Equal(
				t,
				"unexpected value type, "+
					"expected: Input; "+
					`received(json): 1`,
				err.Error(),
			)
		},
	}),
	decl.New(TestError{
		Src:      `query ($v:Int) { f(a:$v) }`,
		VarsJSON: `{"v":{"foo":"bar","baz":42}}`,
		Check: func(t *testing.T, err error) {
			require.Error(t, err)
			require.IsType(t, &gqlparse.ErrorUnexpValType{}, err)
			require.Equal(
				t,
				"unexpected value type, "+
					"expected: Int; "+
					`received(json): {"foo":"bar","baz":42}`,
				err.Error(),
			)
		},
	}),
	decl.New(TestError{
		Src:      `query ($v:Int) { f(a:$v) }`,
		VarsJSON: `{"v":[]}`,
		Check: func(t *testing.T, err error) {
			require.Error(t, err)
			require.IsType(t, &gqlparse.ErrorUnexpValType{}, err)
			require.Equal(
				t,
				"unexpected value type, "+
					"expected: Int; "+
					`received(json): []`,
				err.Error(),
			)
		},
	}),
	decl.New(TestError{
		Src:      `query ($v:Int) { f(a:$v) }`,
		VarsJSON: `{"v":[{"x":2,"y":4}, null, {"x":8,"y":8}]}`,
		Check: func(t *testing.T, err error) {
			require.Error(t, err)
			require.IsType(t, &gqlparse.ErrorUnexpValType{}, err)
			require.Equal(
				t,
				"unexpected value type, "+
					"expected: Int; "+
					`received(json): [{"x":2,"y":4}, null, {"x":8,"y":8}]`,
				err.Error(),
			)
		},
	}),
	decl.New(TestError{
		Src:      `query ($v:Int!) { f(a:$v) }`,
		VarsJSON: `{"v":null}`,
		Check: func(t *testing.T, err error) {
			require.Error(t, err)
			require.IsType(t, &gqlparse.ErrorUnexpValType{}, err)
			require.Equal(
				t,
				"unexpected value type, "+
					"expected: Int!; "+
					`received(json): null`,
				err.Error(),
			)
		},
	}),

	// Invalid variables JSON (non-object)
	decl.New(TestError{
		Src:      `query ($v:String) { f(a:$v) }`,
		VarsJSON: `["okay"]`,
		Check: func(t *testing.T, err error) {
			require.Error(t, err)
			require.IsType(t, &gqlparse.ErrorVarJSONNotObj{}, err)
			require.Equal(t, `expected JSON object for variables, `+
				`received: ["okay"]`, err.Error())
		},
	}),
	decl.New(TestError{
		Src:      `query ($v:Int!) { f(a:$v) }`,
		VarsJSON: `42`,
		Check: func(t *testing.T, err error) {
			require.Error(t, err)
			require.IsType(t, &gqlparse.ErrorVarJSONNotObj{}, err)
			require.Equal(t, `expected JSON object for variables, `+
				`received: 42`, err.Error())
		},
	}),

	// Query syntax error
	decl.New(TestError{
		Src: `query ($v:String = ) { f(a:$v) }`,
		Check: func(t *testing.T, err error) {
			require.Error(t, err)
			require.IsType(t, &gqlparse.ErrorSyntax{}, err)
			require.Equal(t, `syntax error: error at index 19 (')'):`+
				` unexpected token; expected enum value`, err.Error())
		},
	}),

	// Invalid variables JSON (syntax error)
	decl.New(TestError{
		Src:      `query ($v:String) { f(a:$v) }`,
		VarsJSON: `{"v":"first" "missing-comma": "second"}`,
		Check: func(t *testing.T, err error) {
			require.Error(t, err)
			require.IsType(t, &gqlparse.ErrorVarJSONSyntax{}, err)
			require.Equal(t, `variables JSON syntax error`, err.Error())
		},
	}),
	decl.New(TestError{
		Src:      `query ($v:Int!) { f(a:$v) }`,
		VarsJSON: `{v:42}`,
		Check: func(t *testing.T, err error) {
			require.Error(t, err)
			require.IsType(t, &gqlparse.ErrorVarJSONSyntax{}, err)
			require.Equal(t, `variables JSON syntax error`, err.Error())
		},
	}),
	decl.New(TestError{
		Src:      `query ($v:String!) { f(a:$v) }`,
		VarsJSON: `{"v": missing_quotes}`,
		Check: func(t *testing.T, err error) {
			require.Error(t, err)
			require.IsType(t, &gqlparse.ErrorVarJSONSyntax{}, err)
			require.Equal(t, `variables JSON syntax error`, err.Error())
		},
	}),

	// Undeclared variable
	decl.New(TestError{
		Src:      `{ f(a:$u) }`,
		VarsJSON: `{"u":42}`,
		Check: func(t *testing.T, err error) {
			require.Error(t, err)
			require.IsType(t, &gqlparse.ErrorVarUndeclared{}, err)
			require.Equal(t, `variable "u" undeclared`, err.Error())
		},
	}),
	decl.New(TestError{
		Src:      `query ($v:String!) { f(a:$u) }`,
		VarsJSON: `{"v":"okay","u":42}`,
		Check: func(t *testing.T, err error) {
			require.Error(t, err)
			require.IsType(t, &gqlparse.ErrorVarUndeclared{}, err)
			require.Equal(t, `variable "u" undeclared`, err.Error())
		},
	}),

	// Undefined variable value
	decl.New(TestError{
		Src: `query ($v:String!) { f(a:$v) }`,
		Check: func(t *testing.T, err error) {
			require.Error(t, err)
			require.IsType(t, &gqlparse.ErrorVarUndefined{}, err)
			require.Equal(t, `variable "v" undefined`, err.Error())
		},
	}),

	// Fragment undefined
	decl.New(TestError{
		Src: `{ ...f }`,
		Check: func(t *testing.T, err error) {
			require.Error(t, err)
			require.Equal(t, &gqlparse.ErrorFragUndefined{
				FragmentName: []byte("f"),
			}, err)
			require.Equal(t, `fragment "f" undefined`, err.Error())
		},
	}),
	decl.New(TestError{
		Src: `{ ...f }, fragment f on Query { ...x }`,
		Check: func(t *testing.T, err error) {
			require.Error(t, err)
			require.Equal(t, &gqlparse.ErrorFragUndefined{
				FragmentName: []byte("x"),
			}, err)
			require.Equal(t, `fragment "x" undefined`, err.Error())
		},
	}),

	// Fragment limit exceeded
	decl.New(TestError{
		Src: `{
			...f0, ...f1, ...f2, ...f3, ...f4,
			...f5, ...f6, ...f7, ...f8, ...f9,
			...f10, ...f11, ...f12, ...f13, ...f14,
			...f15, ...f16, ...f17, ...f18, ...f19,
			...f20, ...f21, ...f22, ...f23, ...f24,
			...f25, ...f26, ...f27, ...f28, ...f29,
			...f30, ...f31, ...f32, ...f33, ...f34,
			...f35, ...f36, ...f37, ...f38, ...f39,
			...f40, ...f41, ...f42, ...f43, ...f44,
			...f45, ...f46, ...f47, ...f48, ...f49,
			...f50, ...f51, ...f52, ...f53, ...f54,
			...f55, ...f56, ...f57, ...f58, ...f59,
			...f60, ...f61, ...f62, ...f63, ...f64,
			...f65, ...f66, ...f67, ...f68, ...f69,
			...f70, ...f71, ...f72, ...f73, ...f74,
			...f75, ...f76, ...f77, ...f78, ...f79,
			...f80, ...f81, ...f82, ...f83, ...f84,
			...f85, ...f86, ...f87, ...f88, ...f89,
			...f90, ...f91, ...f92, ...f93, ...f94,
			...f95, ...f96, ...f97, ...f98, ...f99,
			...f100, ...f101, ...f102, ...f103, ...f104,
			...f105, ...f106, ...f107, ...f108, ...f109,
			...f110, ...f111, ...f112, ...f113, ...f114,
			...f115, ...f116, ...f117, ...f118, ...f119,
			...f120, ...f121, ...f122, ...f123, ...f124,
			...f125, ...f126, ...f127, ...f128,
		}
		
		fragment f0 on Query { x }
		fragment f1 on Query { x }
		fragment f2 on Query { x }
		fragment f3 on Query { x }
		fragment f4 on Query { x }
		fragment f5 on Query { x }
		fragment f6 on Query { x }
		fragment f7 on Query { x }
		fragment f8 on Query { x }
		fragment f9 on Query { x }
		fragment f10 on Query { x }
		fragment f11 on Query { x }
		fragment f12 on Query { x }
		fragment f13 on Query { x }
		fragment f14 on Query { x }
		fragment f15 on Query { x }
		fragment f16 on Query { x }
		fragment f17 on Query { x }
		fragment f18 on Query { x }
		fragment f19 on Query { x }
		fragment f20 on Query { x }
		fragment f21 on Query { x }
		fragment f22 on Query { x }
		fragment f23 on Query { x }
		fragment f24 on Query { x }
		fragment f25 on Query { x }
		fragment f26 on Query { x }
		fragment f27 on Query { x }
		fragment f28 on Query { x }
		fragment f29 on Query { x }
		fragment f30 on Query { x }
		fragment f31 on Query { x }
		fragment f32 on Query { x }
		fragment f33 on Query { x }
		fragment f34 on Query { x }
		fragment f35 on Query { x }
		fragment f36 on Query { x }
		fragment f37 on Query { x }
		fragment f38 on Query { x }
		fragment f39 on Query { x }
		fragment f40 on Query { x }
		fragment f41 on Query { x }
		fragment f42 on Query { x }
		fragment f43 on Query { x }
		fragment f44 on Query { x }
		fragment f45 on Query { x }
		fragment f46 on Query { x }
		fragment f47 on Query { x }
		fragment f48 on Query { x }
		fragment f49 on Query { x }
		fragment f50 on Query { x }
		fragment f51 on Query { x }
		fragment f52 on Query { x }
		fragment f53 on Query { x }
		fragment f54 on Query { x }
		fragment f55 on Query { x }
		fragment f56 on Query { x }
		fragment f57 on Query { x }
		fragment f58 on Query { x }
		fragment f59 on Query { x }
		fragment f60 on Query { x }
		fragment f61 on Query { x }
		fragment f62 on Query { x }
		fragment f63 on Query { x }
		fragment f64 on Query { x }
		fragment f65 on Query { x }
		fragment f66 on Query { x }
		fragment f67 on Query { x }
		fragment f68 on Query { x }
		fragment f69 on Query { x }
		fragment f70 on Query { x }
		fragment f71 on Query { x }
		fragment f72 on Query { x }
		fragment f73 on Query { x }
		fragment f74 on Query { x }
		fragment f75 on Query { x }
		fragment f76 on Query { x }
		fragment f77 on Query { x }
		fragment f78 on Query { x }
		fragment f79 on Query { x }
		fragment f80 on Query { x }
		fragment f81 on Query { x }
		fragment f82 on Query { x }
		fragment f83 on Query { x }
		fragment f84 on Query { x }
		fragment f85 on Query { x }
		fragment f86 on Query { x }
		fragment f87 on Query { x }
		fragment f88 on Query { x }
		fragment f89 on Query { x }
		fragment f90 on Query { x }
		fragment f91 on Query { x }
		fragment f92 on Query { x }
		fragment f93 on Query { x }
		fragment f94 on Query { x }
		fragment f95 on Query { x }
		fragment f96 on Query { x }
		fragment f97 on Query { x }
		fragment f98 on Query { x }
		fragment f99 on Query { x }
		fragment f100 on Query { x }
		fragment f101 on Query { x }
		fragment f102 on Query { x }
		fragment f103 on Query { x }
		fragment f104 on Query { x }
		fragment f105 on Query { x }
		fragment f106 on Query { x }
		fragment f107 on Query { x }
		fragment f108 on Query { x }
		fragment f109 on Query { x }
		fragment f110 on Query { x }
		fragment f111 on Query { x }
		fragment f112 on Query { x }
		fragment f113 on Query { x }
		fragment f114 on Query { x }
		fragment f115 on Query { x }
		fragment f116 on Query { x }
		fragment f117 on Query { x }
		fragment f118 on Query { x }
		fragment f119 on Query { x }
		fragment f120 on Query { x }
		fragment f121 on Query { x }
		fragment f122 on Query { x }
		fragment f123 on Query { x }
		fragment f124 on Query { x }
		fragment f125 on Query { x }
		fragment f126 on Query { x }
		fragment f127 on Query { x }
		fragment f128 on Query { x } # limit exceeded`,
		Check: func(t *testing.T, err error) {
			require.Error(t, err)
			require.Equal(t, &gqlparse.ErrorFragLimitExceeded{
				Limit: 128,
			}, err)
			require.Equal(t, "fragment limit (128) exceeded", err.Error())
		},
	}),
}

func TestErr(t *testing.T) {
	for _, td := range testdataErr {
		t.Run(td.Decl, func(t *testing.T) {
			gqlparse.NewParser().Parse(
				[]byte(td.Data.Src),
				[]byte(td.Data.OprName),
				[]byte(td.Data.VarsJSON),
				func(
					varVals [][]gqlparse.Token,
					operation []gqlparse.Token,
					selectionSet []gqlparse.Token,
				) {
					t.Fatal("unexpected success!")
				},
				func(err error) {
					require.Error(t, err)
					td.Data.Check(t, err)
				},
			)
		})
	}
}

type TestSuccess struct {
	Src           string
	VarsJSON      string
	OprName       string
	ExpectVarVals map[string][]gqlparse.Token
	ExpectOpr     []gqlparse.Token
}

func MakeTestSuccess(
	operationName string,
	src string,
	varsJSON string,
	expect ...gqlparse.Token,
) decl.Declaration[TestSuccess] {
	return decl.New(TestSuccess{
		Src:       src,
		VarsJSON:  varsJSON,
		OprName:   operationName,
		ExpectOpr: expect,
	})
}

func Token(t gqlscan.Token, value ...string) gqlparse.Token {
	if len(value) > 1 {
		panic(fmt.Errorf("value must not be longer 1, was: %v", value))
	}
	var v []byte
	if len(value) > 0 {
		v = []byte(value[0])
	}
	return gqlparse.Token{
		ID:    t,
		Value: v,
	}
}

type TestError struct {
	Src      string
	VarsJSON string
	OprName  string
	Check    func(*testing.T, error)
}

type WriteValueTest struct {
	Input  []gqlparse.Token
	Expect string
}

func stringifyToken(t gqlparse.Token) string {
	var b strings.Builder
	if i := t.VariableIndex(); i > -1 {
		b.WriteString("variable value identifier (")
		b.WriteString(strconv.Itoa(i))
		b.WriteString(")")
	} else {
		b.WriteString(fmt.Sprintf("%v", t.ID))
	}
	if t.Value != nil {
		b.WriteString(fmt.Sprintf(" (%q)", string(t.Value)))
	}
	return b.String()
}

type TestWriter struct{ t *testing.T }

func (t *TestWriter) Errorf(format string, v ...any) {
	t.t.Helper()
	t.t.Errorf(format, v...)
}
