package constrcheck

import (
	"github.com/graph-guard/ggproxy/gqlparse"
	"github.com/graph-guard/gqlscan"
	gqlast "github.com/vektah/gqlparser/v2/ast"
)

// isWrongType returns true if the value represented in input
// doesn't correspond to the type defined by expect.
func isWrongType(
	expect *gqlast.Type,
	input []gqlparse.Token,
	schema *gqlast.Schema,
) (bool, []gqlparse.Token) {
	var def *gqlast.Definition
	if expect != nil {
		def = schema.Types[expect.NamedType]
	}

	switch input[0].ID {
	case gqlscan.TokenNull:
		if expect == nil {
			return false, input[1:]
		}
		return expect.NonNull, input[1:]
	case gqlscan.TokenArr:
		if expect != nil {
			if expect.Elem == nil {
				return true, nil
			}
			expect = expect.Elem
		}
		input = input[1:]
		for {
			if input[0].ID == gqlscan.TokenArrEnd {
				input = input[1:]
				return false, input
			}
			var wrongType bool
			wrongType, input = isWrongType(expect, input, schema)
			if wrongType {
				return true, nil
			}
		}
	case gqlscan.TokenTrue, gqlscan.TokenFalse:
		if expect == nil || (def != nil && def.Kind == gqlast.Scalar &&
			def.Name != "ID" &&
			def.Name != "Int" &&
			def.Name != "Float" &&
			def.Name != "String") {
			return false, input[1:]
		}
		return expect.NamedType != "Boolean", input[1:]
	case gqlscan.TokenInt:
		if expect == nil || (def != nil && def.Kind == gqlast.Scalar &&
			def.Name != "ID" &&
			def.Name != "String" &&
			def.Name != "Boolean") {
			return false, input[1:]
		}
		return expect.NamedType != "Int" &&
			expect.NamedType != "Float", input[1:]
	case gqlscan.TokenFloat:
		if expect == nil || (def != nil && def.Kind == gqlast.Scalar &&
			def.Name != "ID" &&
			def.Name != "Int" &&
			def.Name != "String" &&
			def.Name != "Boolean") {
			return false, input[1:]
		}
		return expect.NamedType != "Float", input[1:]
	case gqlscan.TokenStr, gqlscan.TokenStrBlock:
		if expect == nil || (def != nil && def.Kind == gqlast.Scalar &&
			def.Name != "Int" &&
			def.Name != "Float" &&
			def.Name != "Boolean") {
			return false, input[1:]
		}
		return expect.NamedType != "String" &&
			expect.NamedType != "ID", input[1:]
	case gqlscan.TokenEnumVal:
		if def == nil || def.Kind == gqlast.Scalar &&
			def.Name != "ID" &&
			def.Name != "Int" &&
			def.Name != "Float" &&
			def.Name != "String" &&
			def.Name != "Boolean" {
			// No expectation or custom scalar.
			return false, input[1:]
		}
		for i := range def.EnumValues {
			if def.EnumValues[i].Name == string(input[0].Value) {
				return false, input[1:]
			}
		}
		return true, nil
	case gqlscan.TokenObj:
		if def == nil || def.Kind == gqlast.Scalar {
			// No expectation or custom scalar type.
			input = input[1:]
		SKIP_OBJECT:
			for levelObj := 1; ; input = input[1:] {
				switch input[0].ID {
				case gqlscan.TokenObj:
					levelObj++
				case gqlscan.TokenObjEnd:
					levelObj--
					if levelObj < 1 {
						input = input[1:]
						break SKIP_OBJECT
					}
				}
			}
			return false, input
		} else if def.Kind != gqlast.InputObject {
			return true, nil
		}

		input = input[1:]
	SCAN_OBJECT:
		for len(input) > 0 {
			if input[0].ID == gqlscan.TokenObjEnd {
				input = input[1:]
				break SCAN_OBJECT
			}

			field := input[0]
			for i := range def.Fields {
				if def.Fields[i].Name == string(field.Value) {
					// Check field value
					var wrongType bool
					wrongType, input = isWrongType(
						def.Fields[i].Type, input[1:], schema,
					)
					if wrongType {
						return true, nil
					}
					continue SCAN_OBJECT
				}
			}
			// Field not found in expected input object type
			return true, nil
		}
		return false, input
	}

	return true, nil
}
