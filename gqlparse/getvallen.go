package gqlparse

import "github.com/graph-guard/gqlscan"

// GetValLen reads the length in tokens from the first value in t.
// For example:
//
//	`[1,2,3]` has a length of 5 tokens
//	`{x:""}` has a length of 4 tokens
//	`null` has a length of 1 tokens
func GetValLen(t []Token) (length int) {
	switch t[0].ID {
	case gqlscan.TokenNull, gqlscan.TokenInt, gqlscan.TokenFloat,
		gqlscan.TokenStr, gqlscan.TokenStrBlock, gqlscan.TokenEnumVal,
		gqlscan.TokenTrue, gqlscan.TokenFalse:
		return 1
	case gqlscan.TokenArr:
		t = t[1:]
		length += 1
	SCAN_ARR:
		for levelArr := 1; ; t = t[1:] {
			length++
			switch t[0].ID {
			case gqlscan.TokenArr:
				levelArr++
			case gqlscan.TokenArrEnd:
				levelArr--
				if levelArr < 1 {
					break SCAN_ARR
				}
			}
		}
	case gqlscan.TokenObj:
		t = t[1:]
		length += 1
	SCAN_OBJ:
		for levelObj := 1; ; t = t[1:] {
			length++
			switch t[0].ID {
			case gqlscan.TokenObj:
				levelObj++
			case gqlscan.TokenObjEnd:
				levelObj--
				if levelObj < 1 {
					break SCAN_OBJ
				}
			}
		}
	}
	return length
}
