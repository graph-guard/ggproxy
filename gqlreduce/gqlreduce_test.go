package gqlreduce_test

import (
	"fmt"
	"testing"

	"github.com/graph-guard/gguard-proxy/gqlreduce"
	"github.com/graph-guard/gguard-proxy/utilities/decl"
	"github.com/graph-guard/gqlscan"
	"github.com/stretchr/testify/require"
)

var testdata = []decl.Declaration[TestSuccess]{
	decl.New(TestSuccess{
		Src: `{x}`,
		Expect: []gqlreduce.Token{
			Token(gqlscan.TokenDefQry),
			Token(gqlscan.TokenSet),
			Token(gqlscan.TokenField, "x"),
			Token(gqlscan.TokenSetEnd),
		},
	}),
	// Inline anonymous fragments
	decl.New(TestSuccess{
		Src: `{...{x}}`,
		Expect: []gqlreduce.Token{
			Token(gqlscan.TokenDefQry),
			Token(gqlscan.TokenSet),
			Token(gqlscan.TokenField, "x"),
			Token(gqlscan.TokenSetEnd),
		},
	}),
	decl.New(TestSuccess{
		Src: `{...{x ...{y}}}`,
		Expect: []gqlreduce.Token{
			Token(gqlscan.TokenDefQry),
			Token(gqlscan.TokenSet),
			Token(gqlscan.TokenField, "x"),
			Token(gqlscan.TokenField, "y"),
			Token(gqlscan.TokenSetEnd),
		},
	}),
	// Inline named fragments
	decl.New(TestSuccess{
		Src: `{...f}, fragment f on Query {x}`,
		Expect: []gqlreduce.Token{
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
		Expect: []gqlreduce.Token{
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
		Expect: []gqlreduce.Token{
			Token(gqlscan.TokenDefQry),
			Token(gqlscan.TokenOprName, "X"),
			Token(gqlscan.TokenSet),

			Token(gqlscan.TokenField, "foo"),
			Token(gqlscan.TokenArgList),
			Token(gqlscan.TokenArgName, "bar"),
			Token(gqlscan.TokenStr, "text"),
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
		Expect: []gqlreduce.Token{
			Token(gqlscan.TokenDefQry),
			Token(gqlscan.TokenOprName, "X"),
			Token(gqlscan.TokenSet),

			Token(gqlscan.TokenField, "foo"),
			Token(gqlscan.TokenArgList),
			Token(gqlscan.TokenArgName, "bar"),
			Token(gqlscan.TokenStr, "text"),
			Token(gqlscan.TokenArgListEnd),

			Token(gqlscan.TokenField, "bar"),
			Token(gqlscan.TokenArgList),
			Token(gqlscan.TokenArgName, "baz"),
			Token(gqlscan.TokenStr, "text"),
			Token(gqlscan.TokenArgListEnd),

			Token(gqlscan.TokenField, "bar"),
			Token(gqlscan.TokenArgList),
			Token(gqlscan.TokenArgName, "baz"),
			Token(gqlscan.TokenStr, "text"),
			Token(gqlscan.TokenArgListEnd),

			Token(gqlscan.TokenField, "baz"),
			Token(gqlscan.TokenArgList),
			Token(gqlscan.TokenArgName, "fuz"),
			Token(gqlscan.TokenStr, "text"),
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
		Expect: []gqlreduce.Token{
			Token(gqlscan.TokenDefQry),
			Token(gqlscan.TokenOprName, "Q"),
			Token(gqlscan.TokenSet),
			Token(gqlscan.TokenFragInline, "Query"),
			Token(gqlscan.TokenSet),
			Token(gqlscan.TokenField, "f"),
			Token(gqlscan.TokenArgList),

			// $v_s: String! = """default value""",
			Token(gqlscan.TokenArgName, "a1"),
			Token(gqlscan.TokenStr, "from JSON"),

			// $v_i: Int! = 42,
			Token(gqlscan.TokenArgName, "a2"),
			Token(gqlscan.TokenInt, "10042"),

			// $v_f: Float! = 3.14,
			Token(gqlscan.TokenArgName, "a3"),
			Token(gqlscan.TokenFloat, "100.314"),

			// $v_b: Boolean! = true,
			Token(gqlscan.TokenArgName, "a4"),
			Token(gqlscan.TokenFalse),

			// $v_d: ID! = "default ID",
			Token(gqlscan.TokenArgName, "a5"),
			Token(gqlscan.TokenStr, "ID from JSON"),

			// $v_o: InputObj! = {foo: "bar"},
			Token(gqlscan.TokenArgName, "a6"),
			Token(gqlscan.TokenObj),
			Token(gqlscan.TokenObjField, "foo"),
			Token(gqlscan.TokenStr, "bar from JSON"),
			Token(gqlscan.TokenObjEnd),

			// $v_so: String = null,
			Token(gqlscan.TokenArgName, "a7"),
			Token(gqlscan.TokenNull),

			// $v_io: Int = null,
			Token(gqlscan.TokenArgName, "a8"),
			Token(gqlscan.TokenNull),

			// $v_fo: Float = null,
			Token(gqlscan.TokenArgName, "a9"),
			Token(gqlscan.TokenNull),

			// $v_bo: Boolean = null,
			Token(gqlscan.TokenArgName, "a10"),
			Token(gqlscan.TokenNull),

			// $v_do: ID = null,
			Token(gqlscan.TokenArgName, "a11"),
			Token(gqlscan.TokenNull),

			// $v_oo: InputObj = null,
			Token(gqlscan.TokenArgName, "a12"),
			Token(gqlscan.TokenNull),

			// $v_aon: [String] = null,
			Token(gqlscan.TokenArgName, "a13"),
			Token(gqlscan.TokenNull),

			// $v_aoy: [String] = [],
			Token(gqlscan.TokenArgName, "a14"),
			Token(gqlscan.TokenArr),
			Token(gqlscan.TokenArrEnd),

			// $v_a_so: [String]! = ["okay", null],
			Token(gqlscan.TokenArgName, "a15"),
			Token(gqlscan.TokenArr),
			Token(gqlscan.TokenStr, "okay"),
			Token(gqlscan.TokenNull),
			Token(gqlscan.TokenArrEnd),

			// $v_a_ao_so: [[String]]! = [["okay", null], [], null],
			Token(gqlscan.TokenArgName, "a16"),
			Token(gqlscan.TokenArr),
			Token(gqlscan.TokenArr),
			Token(gqlscan.TokenStr, "okay"),
			Token(gqlscan.TokenNull),
			Token(gqlscan.TokenArrEnd),
			Token(gqlscan.TokenArr),
			Token(gqlscan.TokenArrEnd),
			Token(gqlscan.TokenNull),
			Token(gqlscan.TokenArrEnd),

			// $v_a_io: [InputObj]! = [{a: "1", b: null, c: 42, d: false}, null],
			Token(gqlscan.TokenArgName, "a17"),
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
		Expect: []gqlreduce.Token{
			Token(gqlscan.TokenDefQry),
			Token(gqlscan.TokenOprName, "Q"),
			Token(gqlscan.TokenSet),
			Token(gqlscan.TokenFragInline, "Query"),
			Token(gqlscan.TokenSet),
			Token(gqlscan.TokenField, "f"),
			Token(gqlscan.TokenArgList),

			// $v_s: String! = """default value""",
			Token(gqlscan.TokenArgName, "a1"),
			Token(gqlscan.TokenStrBlock, "default value"),

			// $v_i: Int! = 42,
			Token(gqlscan.TokenArgName, "a2"),
			Token(gqlscan.TokenInt, "42"),

			// $v_f: Float! = 3.14,
			Token(gqlscan.TokenArgName, "a3"),
			Token(gqlscan.TokenFloat, "3.14"),

			// $v_b: Boolean! = true,
			Token(gqlscan.TokenArgName, "a4"),
			Token(gqlscan.TokenTrue),

			// $v_d: ID! = "default ID",
			Token(gqlscan.TokenArgName, "a5"),
			Token(gqlscan.TokenStr, "default ID"),

			// $v_o: InputObj! = {foo: "bar"},
			Token(gqlscan.TokenArgName, "a6"),
			Token(gqlscan.TokenObj),
			Token(gqlscan.TokenObjField, "foo"),
			Token(gqlscan.TokenStr, "bar"),
			Token(gqlscan.TokenObjEnd),

			// $v_so: String = null,
			Token(gqlscan.TokenArgName, "a7"),
			Token(gqlscan.TokenNull),

			// $v_io: Int = null,
			Token(gqlscan.TokenArgName, "a8"),
			Token(gqlscan.TokenNull),

			// $v_fo: Float = null,
			Token(gqlscan.TokenArgName, "a9"),
			Token(gqlscan.TokenNull),

			// $v_bo: Boolean = null,
			Token(gqlscan.TokenArgName, "a10"),
			Token(gqlscan.TokenNull),

			// $v_do: ID = null,
			Token(gqlscan.TokenArgName, "a11"),
			Token(gqlscan.TokenNull),

			// $v_oo: InputObj = null,
			Token(gqlscan.TokenArgName, "a12"),
			Token(gqlscan.TokenNull),

			// $v_aon: [String] = null,
			Token(gqlscan.TokenArgName, "a13"),
			Token(gqlscan.TokenNull),

			// $v_aoy: [String] = [],
			Token(gqlscan.TokenArgName, "a14"),
			Token(gqlscan.TokenArr),
			Token(gqlscan.TokenArrEnd),

			// $v_a_so: [String]! = ["okay", null],
			Token(gqlscan.TokenArgName, "a15"),
			Token(gqlscan.TokenArr),
			Token(gqlscan.TokenStr, "okay"),
			Token(gqlscan.TokenNull),
			Token(gqlscan.TokenArrEnd),

			// $v_a_ao_so: [[String]]! = [["okay", null], [], null],
			Token(gqlscan.TokenArgName, "a16"),
			Token(gqlscan.TokenArr),
			Token(gqlscan.TokenArr),
			Token(gqlscan.TokenStr, "okay"),
			Token(gqlscan.TokenNull),
			Token(gqlscan.TokenArrEnd),
			Token(gqlscan.TokenArr),
			Token(gqlscan.TokenArrEnd),
			Token(gqlscan.TokenNull),
			Token(gqlscan.TokenArrEnd),

			// $v_a_io: [InputObj]! = [{a: "1", b: null, c: 42, d: false}, null],
			Token(gqlscan.TokenArgName, "a17"),
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

			Token(gqlscan.TokenArgListEnd),
			Token(gqlscan.TokenSetEnd),
			Token(gqlscan.TokenSetEnd),
		},
	}),
}

func TestOK(t *testing.T) {
	for _, td := range testdata {
		t.Run(td.Decl, func(t *testing.T) {
			var expectOriginal []gqlreduce.Token
			err := gqlscan.ScanAll([]byte(td.Data.Src), func(i *gqlscan.Iterator) {
				expectOriginal = append(expectOriginal, gqlreduce.Token{
					Type:  i.Token(),
					Value: i.Value(),
				})
			})
			require.False(t, err.IsErr())

			gqlreduce.NewReducer().Reduce(
				[]byte(td.Data.Src),
				[]byte(td.Data.OprName),
				[]byte(td.Data.VarsJSON),
				func(reduced []gqlreduce.Token) {
					// fmt.Printf("expected: (%d)\n", len(td.Expected))
					// for i, x := range td.Expected {
					// 	fmt.Printf(" %d: %v %q\n", i, x.Type, string(x.Value))
					// }
					// fmt.Println(" ")
					// fmt.Printf("reduced: (%d)\n", len(reduced))
					// for i, x := range reduced {
					// 	fmt.Printf(" %d: %v %q\n", i, x.Type, string(x.Value))
					// }
					// fmt.Println(" ")

					require.Equal(t, td.Data.Expect, reduced)
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
			require.Equal(t, &gqlreduce.ErrorOprNotFound{
				OperationName: []byte(""),
			}, err)
			require.Equal(t, `operation "" not found`, err.Error())
		},
	}),
	decl.New(TestError{
		Src:     `query A {x}, query B {x}`,
		OprName: "C",
		Check: func(t *testing.T, err error) {
			require.Equal(t, &gqlreduce.ErrorOprNotFound{
				OperationName: []byte("C"),
			}, err)
			require.Equal(t, `operation "C" not found`, err.Error())
		},
	}),
	decl.New(TestError{
		Src:     `{x}`,
		OprName: "A",
		Check: func(t *testing.T, err error) {
			require.Equal(t, &gqlreduce.ErrorOprNotFound{
				OperationName: []byte("A"),
			}, err)
			require.Equal(t, `operation "A" not found`, err.Error())
		},
	}),

	// Non-exclusive anonymous operation
	decl.New(TestError{
		Src: `{x}, query{x}`,
		Check: func(t *testing.T, err error) {
			require.Equal(t, &gqlreduce.ErrorOprAnonNonExcl{}, err)
			require.Equal(t, `non-exclusive anonymous operation`, err.Error())
		},
	}),
	decl.New(TestError{
		Src: `{x}, mutation M {x}`,
		Check: func(t *testing.T, err error) {
			require.Equal(t, &gqlreduce.ErrorOprAnonNonExcl{}, err)
			require.Equal(t, `non-exclusive anonymous operation`, err.Error())
		},
	}),
	decl.New(TestError{
		Src: `query A {x}, query {x}`,
		Check: func(t *testing.T, err error) {
			require.Equal(t, &gqlreduce.ErrorOprAnonNonExcl{}, err)
			require.Equal(t, `non-exclusive anonymous operation`, err.Error())
		},
	}),

	// Redeclared operation
	decl.New(TestError{
		Src: `query A {x}, query A {x}`,
		Check: func(t *testing.T, err error) {
			require.Equal(t, &gqlreduce.ErrorRedecOpr{
				OperationName: []byte("A"),
			}, err)
			require.Equal(t, `operation "A" redeclared`, err.Error())
		},
	}),
	decl.New(TestError{
		Src: `query M {x}, mutation M {x}`,
		Check: func(t *testing.T, err error) {
			require.Equal(t, &gqlreduce.ErrorRedecOpr{
				OperationName: []byte("M"),
			}, err)
			require.Equal(t, `operation "M" redeclared`, err.Error())
		},
	}),
	decl.New(TestError{
		Src: `query S {x}, subscription S {x}`,
		Check: func(t *testing.T, err error) {
			require.Equal(t, &gqlreduce.ErrorRedecOpr{
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
			require.Equal(t, &gqlreduce.ErrorRedecFrag{
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
			require.Equal(t, &gqlreduce.ErrorFragUnused{
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
			require.Equal(t, &gqlreduce.ErrorFragRecurse{
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
			require.Equal(t, &gqlreduce.ErrorFragRecurse{
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
			require.Equal(t, &gqlreduce.ErrorFragRecurse{
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
			require.Equal(t, &gqlreduce.ErrorRedeclVar{
				VariableName: []byte("v1"),
			}, err)
			require.Equal(t, `variable "v1" redeclared`, err.Error())
		},
	}),
	decl.New(TestError{
		Src:     `query Q ($v2:String, $v2:Int){f(a:$v2)}`,
		OprName: "Q",
		Check: func(t *testing.T, err error) {
			require.Equal(t, &gqlreduce.ErrorRedeclVar{
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
			require.IsType(t, &gqlreduce.ErrorUnexpValType{}, err)
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
			require.IsType(t, &gqlreduce.ErrorUnexpValType{}, err)
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
			require.IsType(t, &gqlreduce.ErrorUnexpValType{}, err)
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
			require.IsType(t, &gqlreduce.ErrorUnexpValType{}, err)
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
			require.IsType(t, &gqlreduce.ErrorUnexpValType{}, err)
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
			require.IsType(t, &gqlreduce.ErrorUnexpValType{}, err)
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
			require.IsType(t, &gqlreduce.ErrorUnexpValType{}, err)
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
			require.IsType(t, &gqlreduce.ErrorUnexpValType{}, err)
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
			require.IsType(t, &gqlreduce.ErrorUnexpValType{}, err)
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
			require.IsType(t, &gqlreduce.ErrorUnexpValType{}, err)
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
			require.IsType(t, &gqlreduce.ErrorUnexpValType{}, err)
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
			require.IsType(t, &gqlreduce.ErrorUnexpValType{}, err)
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
			require.IsType(t, &gqlreduce.ErrorUnexpValType{}, err)
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
			require.IsType(t, &gqlreduce.ErrorUnexpValType{}, err)
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
			require.IsType(t, &gqlreduce.ErrorUnexpValType{}, err)
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
			require.IsType(t, &gqlreduce.ErrorUnexpValType{}, err)
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
			require.IsType(t, &gqlreduce.ErrorUnexpValType{}, err)
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
			require.IsType(t, &gqlreduce.ErrorUnexpValType{}, err)
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
			require.IsType(t, &gqlreduce.ErrorUnexpValType{}, err)
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
			require.IsType(t, &gqlreduce.ErrorUnexpValType{}, err)
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
			require.IsType(t, &gqlreduce.ErrorVarJSONNotObj{}, err)
			require.Equal(t, `expected JSON object for variables, `+
				`received: ["okay"]`, err.Error())
		},
	}),
	decl.New(TestError{
		Src:      `query ($v:Int!) { f(a:$v) }`,
		VarsJSON: `42`,
		Check: func(t *testing.T, err error) {
			require.Error(t, err)
			require.IsType(t, &gqlreduce.ErrorVarJSONNotObj{}, err)
			require.Equal(t, `expected JSON object for variables, `+
				`received: 42`, err.Error())
		},
	}),

	// Query syntax error
	decl.New(TestError{
		Src: `query ($v:String = ) { f(a:$v) }`,
		Check: func(t *testing.T, err error) {
			require.Error(t, err)
			require.IsType(t, &gqlreduce.ErrorSyntax{}, err)
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
			require.IsType(t, &gqlreduce.ErrorVarJSONSyntax{}, err)
			require.Equal(t, `variables JSON syntax error`, err.Error())
		},
	}),
	decl.New(TestError{
		Src:      `query ($v:Int!) { f(a:$v) }`,
		VarsJSON: `{v:42}`,
		Check: func(t *testing.T, err error) {
			require.Error(t, err)
			require.IsType(t, &gqlreduce.ErrorVarJSONSyntax{}, err)
			require.Equal(t, `variables JSON syntax error`, err.Error())
		},
	}),
	decl.New(TestError{
		Src:      `query ($v:String!) { f(a:$v) }`,
		VarsJSON: `{"v": missing_quotes}`,
		Check: func(t *testing.T, err error) {
			require.Error(t, err)
			require.IsType(t, &gqlreduce.ErrorVarJSONSyntax{}, err)
			require.Equal(t, `variables JSON syntax error`, err.Error())
		},
	}),

	// Undeclared variable
	decl.New(TestError{
		Src:      `{ f(a:$u) }`,
		VarsJSON: `{"u":42}`,
		Check: func(t *testing.T, err error) {
			require.Error(t, err)
			require.IsType(t, &gqlreduce.ErrorVarUndeclared{}, err)
			require.Equal(t, `variable "u" undeclared`, err.Error())
		},
	}),
	decl.New(TestError{
		Src:      `query ($v:String!) { f(a:$u) }`,
		VarsJSON: `{"v":"okay","u":42}`,
		Check: func(t *testing.T, err error) {
			require.Error(t, err)
			require.IsType(t, &gqlreduce.ErrorVarUndeclared{}, err)
			require.Equal(t, `variable "u" undeclared`, err.Error())
		},
	}),

	// Undefined variable value
	decl.New(TestError{
		Src: `query ($v:String!) { f(a:$v) }`,
		Check: func(t *testing.T, err error) {
			require.Error(t, err)
			require.IsType(t, &gqlreduce.ErrorVarUndefined{}, err)
			require.Equal(t, `variable "v" undefined`, err.Error())
		},
	}),

	// Fragment undefined
	decl.New(TestError{
		Src: `{ ...f }`,
		Check: func(t *testing.T, err error) {
			require.Error(t, err)
			require.Equal(t, &gqlreduce.ErrorFragUndefined{
				FragmentName: []byte("f"),
			}, err)
			require.Equal(t, `fragment "f" undefined`, err.Error())
		},
	}),
	decl.New(TestError{
		Src: `{ ...f }, fragment f on Query { ...x }`,
		Check: func(t *testing.T, err error) {
			require.Error(t, err)
			require.Equal(t, &gqlreduce.ErrorFragUndefined{
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
			require.Equal(t, &gqlreduce.ErrorFragLimitExceeded{
				Limit: 128,
			}, err)
			require.Equal(t, "fragment limit (128) exceeded", err.Error())
		},
	}),
}

func TestErr(t *testing.T) {
	for _, td := range testdataErr {
		t.Run(td.Decl, func(t *testing.T) {
			gqlreduce.NewReducer().Reduce(
				[]byte(td.Data.Src),
				[]byte(td.Data.OprName),
				[]byte(td.Data.VarsJSON),
				func(reduced []gqlreduce.Token) {
					t.Fatal("this function is expected not to be called!")
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
	Src      string
	VarsJSON string
	OprName  string
	Expect   []gqlreduce.Token
}

func MakeTestSuccess(
	operationName string,
	src string,
	varsJSON string,
	expect ...gqlreduce.Token,
) decl.Declaration[TestSuccess] {
	return decl.New(TestSuccess{
		Src:      src,
		VarsJSON: varsJSON,
		OprName:  operationName,
		Expect:   expect,
	})
}

func Token(t gqlscan.Token, value ...string) gqlreduce.Token {
	if len(value) > 1 {
		panic(fmt.Errorf("value must not be longer 1, was: %v", value))
	}
	var v []byte
	if len(value) > 0 {
		v = []byte(value[0])
	}
	return gqlreduce.Token{
		Type:  t,
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
	Input  []gqlreduce.Token
	Expect string
}