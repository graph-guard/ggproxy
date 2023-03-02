package constrcheck

import (
	"github.com/graph-guard/ggproxy/engine/playmon/internal/tokenreader"
	"github.com/graph-guard/gqlscan"
	gqlast "github.com/vektah/gqlparser/v2/ast"
)

// isWrongType returns true if the value represented in input
// doesn't correspond to the type defined by expect.
func isWrongType(
	r *tokenreader.Reader,
	expect *gqlast.Type,
	schema *gqlast.Schema,
) bool {
	var def *gqlast.Definition
	if expect != nil {
		def = schema.Types[expect.NamedType]
	}

	switch read := r.ReadOne(); read.ID {
	case gqlscan.TokenNull:
		if expect == nil {
			return false
		}
		return expect.NonNull
	case gqlscan.TokenArr:
		if expect != nil {
			if expect.Elem == nil {
				return true
			}
			expect = expect.Elem
		}
		for {
			rBefore := r
			if r.ReadOne().ID == gqlscan.TokenArrEnd {
				return false
			}
			if isWrongType(rBefore, expect, schema) {
				return true
			}
		}
	case gqlscan.TokenTrue, gqlscan.TokenFalse:
		if expect == nil || (def != nil && def.Kind == gqlast.Scalar &&
			def.Name != "ID" &&
			def.Name != "Int" &&
			def.Name != "Float" &&
			def.Name != "String") {
			return false
		}
		return expect.NamedType != "Boolean"
	case gqlscan.TokenInt:
		if expect == nil || (def != nil && def.Kind == gqlast.Scalar &&
			def.Name != "ID" &&
			def.Name != "String" &&
			def.Name != "Boolean") {
			return false
		}
		return expect.NamedType != "Int" &&
			expect.NamedType != "Float"
	case gqlscan.TokenFloat:
		if expect == nil || (def != nil && def.Kind == gqlast.Scalar &&
			def.Name != "ID" &&
			def.Name != "Int" &&
			def.Name != "String" &&
			def.Name != "Boolean") {
			return false
		}
		return expect.NamedType != "Float"
	case gqlscan.TokenStr, gqlscan.TokenStrBlock:
		if expect == nil || (def != nil && def.Kind == gqlast.Scalar &&
			def.Name != "Int" &&
			def.Name != "Float" &&
			def.Name != "Boolean") {
			return false
		}
		return expect.NamedType != "String" &&
			expect.NamedType != "ID"
	case gqlscan.TokenEnumVal:
		if def == nil || def.Kind == gqlast.Scalar &&
			def.Name != "ID" &&
			def.Name != "Int" &&
			def.Name != "Float" &&
			def.Name != "String" &&
			def.Name != "Boolean" {
			// No expectation or custom scalar.
			return false
		}
		for i := range def.EnumValues {
			if def.EnumValues[i].Name == string(read.Value) {
				return false
			}
		}
		return true
	case gqlscan.TokenObj:
		if def == nil || def.Kind == gqlast.Scalar {
			// No expectation or custom scalar type.
			// r.ReadOne()
		SKIP_OBJECT:
			for levelObj := 1; ; {
				switch r.ReadOne().ID {
				case gqlscan.TokenObj:
					levelObj++
				case gqlscan.TokenObjEnd:
					levelObj--
					if levelObj < 1 {
						break SKIP_OBJECT
					}
				}
			}
			return false
		} else if def.Kind != gqlast.InputObject {
			return true
		}
	SCAN_OBJECT:
		for !r.EOF() {
			if read = r.ReadOne(); read.ID == gqlscan.TokenObjEnd {
				break SCAN_OBJECT
			}

			for i := range def.Fields {
				if def.Fields[i].Name == string(read.Value) {
					// Check field value
					if isWrongType(r, def.Fields[i].Type, schema) {
						return true
					}
					continue SCAN_OBJECT
				}
			}
			// Field not found in expected input object type
			return true
		}
		return false
	}
	return true
}
