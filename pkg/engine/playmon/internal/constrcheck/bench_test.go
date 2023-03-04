package constrcheck_test

// import (
// 	"strconv"
// 	"strings"
// 	"testing"

// 	"github.com/graph-guard/ggproxy/pkg/engine/playmon/internal/constrcheck"
// 	"github.com/graph-guard/ggproxy/pkg/gqlparse"
// 	"github.com/graph-guard/gqlscan"
// 	"github.com/graph-guard/gqt/v4"
// 	"github.com/vektah/gqlparser/v2"
// 	"github.com/vektah/gqlparser/v2/ast"
// )

// var GB bool

// func Token(t gqlscan.Token, value string) gqlparse.Token {
// 	v := []byte(nil)
// 	if value != "" {
// 		v = []byte(value)
// 	}
// 	return gqlparse.Token{ID: t, Value: v}
// }

// func BenchmarkSimple(b *testing.B) {
// 	fn := makeBenchmarkFn(
// 		b,
// 		`type Query { f(a:Int!):Int! }`,
// 		`query { f(a:42) }`,
// 		"Query.f|a",
// 		Token(gqlscan.TokenInt, "42"),
// 	)

// 	b.ResetTimer()
// 	for n := 0; n < b.N; n++ {
// 		if GB = fn(); !GB {
// 			b.Fatal("unexpected result: ", GB)
// 		}
// 	}
// }

// func BenchmarkSimpleArray(b *testing.B) {
// 	fn := makeBenchmarkFn(
// 		b,
// 		`type Query { f(a:[Int!]!):Int! }`,
// 		`query { f(a:[1,2,3,4,5]) }`,
// 		"Query.f|a",
// 		Token(gqlscan.TokenArr, ""),
// 		Token(gqlscan.TokenInt, "1"),
// 		Token(gqlscan.TokenInt, "2"),
// 		Token(gqlscan.TokenInt, "3"),
// 		Token(gqlscan.TokenInt, "4"),
// 		Token(gqlscan.TokenInt, "5"),
// 		Token(gqlscan.TokenArrEnd, ""),
// 	)

// 	b.ResetTimer()
// 	for n := 0; n < b.N; n++ {
// 		if GB = fn(); !GB {
// 			b.Fatal("unexpected result: ", GB)
// 		}
// 	}
// }

// func BenchmarkBigArray(b *testing.B) {
// 	items := 1024

// 	tokens := make([]gqlparse.Token, items+2)
// 	tokens[0] = Token(gqlscan.TokenArr, "")
// 	tokens[len(tokens)-1] = Token(gqlscan.TokenArrEnd, "")
// 	for i := 1; i <= items; i++ {
// 		tokens[i] = Token(gqlscan.TokenInt, strconv.Itoa(i))
// 	}

// 	var tmpl strings.Builder
// 	tmpl.WriteString(`query { f(a:[`)
// 	for i := 1; i <= items; i++ {
// 		tmpl.WriteString(strconv.Itoa(i))
// 		if i != items {
// 			tmpl.WriteByte(',')
// 		}
// 	}
// 	tmpl.WriteString(`]) }`)

// 	fn := makeBenchmarkFn(
// 		b,
// 		`type Query { f(a:[Int!]!):Int! }`,
// 		tmpl.String(),
// 		"Query.f|a",
// 		tokens...,
// 	)

// 	b.ResetTimer()
// 	for n := 0; n < b.N; n++ {
// 		if GB = fn(); !GB {
// 			b.Fatal("unexpected result: ", GB)
// 		}
// 	}
// }

// func BenchmarkObject(b *testing.B) {
// 	fn := makeBenchmarkFn(
// 		b,
// 		`
// 			type Query { f(object:Object!):Int }
// 			input Object { subobject: SubObject! }
// 			input SubObject { array: [ArrayObject!]! }
// 			input ArrayObject { name: String!, index: Int! }
// 		`,
// 		`query { f(object:{
// 			subobject: {
// 				array: [{
// 					name: "first", index: 0
// 				}, {
// 					name: "second", index: 1
// 				}]
// 			}
// 		})}`,
// 		"Query.f|object",
// 		Token(gqlscan.TokenObj, ""),
// 		Token(gqlscan.TokenObjField, "subobject"),
// 		Token(gqlscan.TokenObj, ""),
// 		Token(gqlscan.TokenObjField, "array"),
// 		Token(gqlscan.TokenArr, ""),

// 		Token(gqlscan.TokenObj, ""),
// 		Token(gqlscan.TokenObjField, "name"),
// 		Token(gqlscan.TokenStr, "first"),
// 		Token(gqlscan.TokenObjField, "index"),
// 		Token(gqlscan.TokenInt, "0"),
// 		Token(gqlscan.TokenObjEnd, ""),

// 		Token(gqlscan.TokenObj, ""),
// 		Token(gqlscan.TokenObjField, "name"),
// 		Token(gqlscan.TokenStr, "second"),
// 		Token(gqlscan.TokenObjField, "index"),
// 		Token(gqlscan.TokenInt, "1"),
// 		Token(gqlscan.TokenObjEnd, ""),

// 		Token(gqlscan.TokenArrEnd, ""),
// 		Token(gqlscan.TokenObjEnd, ""),
// 		Token(gqlscan.TokenObjEnd, ""),
// 	)

// 	b.ResetTimer()
// 	for n := 0; n < b.N; n++ {
// 		if GB = fn(); !GB {
// 			b.Fatal("unexpected result: ", GB)
// 		}
// 	}
// }

// func makeBenchmarkFn(
// 	b interface{ Fatal(...any) },
// 	schema, template, path string, tokens ...gqlparse.Token,
// ) func() bool {
// 	s, err := gqlparser.LoadSchema(&ast.Source{
// 		Name:  "schema.graphqls",
// 		Input: schema,
// 	})
// 	if err != nil {
// 		b.Fatal(err)
// 	}

// 	p, err := gqt.NewParser([]gqt.Source{{
// 		Name:    "schema.graphqls",
// 		Content: schema,
// 	}})
// 	if err != nil {
// 		b.Fatal(err)
// 	}

// 	opr, _, errs := p.Parse([]byte(template))
// 	if errs != nil {
// 		b.Fatal(errs)
// 	}

// 	m := constrcheck.New(opr, s)

// 	inputs := map[string][]gqlparse.Token{path: tokens}

// 	m.Init(t)

// 	return func() bool {
// 		return m.Check(path)
// 	}
// }
