package scanval

import (
	"github.com/graph-guard/ggproxy/engine/playmon/internal/tokenreader"
	"github.com/graph-guard/gqlscan"
)

// Length reads the length in tokens from the first value in t.
// For example:
//
//	`[1,2,3]` has a length of 5 tokens
//	`{x:""}` has a length of 4 tokens
//	`null` has a length of 1 tokens
func Length(r *tokenreader.Reader) (length int) {
	switch r.ReadOne().ID {
	case gqlscan.TokenNull, gqlscan.TokenInt, gqlscan.TokenFloat,
		gqlscan.TokenStr, gqlscan.TokenStrBlock, gqlscan.TokenEnumVal,
		gqlscan.TokenTrue, gqlscan.TokenFalse:
		return 1
	case gqlscan.TokenArr:
		length += 1
	SCAN_ARR:
		for levelArr := 1; ; {
			length++
			switch r.ReadOne().ID {
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
		length += 1
	SCAN_OBJ:
		for levelObj := 1; !r.EOF(); {
			length++
			switch r.ReadOne().ID {
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
