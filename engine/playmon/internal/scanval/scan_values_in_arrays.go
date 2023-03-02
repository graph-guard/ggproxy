package scanval

import (
	"github.com/graph-guard/ggproxy/engine/playmon/internal/tokenreader"
	"github.com/graph-guard/gqlscan"
)

// InArrays calls fn for every non-array value contained in x recursively.
// Immediately returns true if any call to fn returned true as well.
func InArrays(
	r *tokenreader.Reader,
	fn func(r *tokenreader.Reader) (stop bool),
) (stopped bool) {
	for levelArr := 0; !r.EOF(); {
		rBefore := *r
		switch r.ReadOne().ID {
		case gqlscan.TokenTrue,
			gqlscan.TokenFalse,
			gqlscan.TokenStr,
			gqlscan.TokenStrBlock,
			gqlscan.TokenFloat,
			gqlscan.TokenInt,
			gqlscan.TokenEnumVal,
			gqlscan.TokenNull:
			rAfter := *r
			*r = rBefore
			if fn(r) {
				return true
			}
			*r = rAfter
		case gqlscan.TokenObj:
		OBJ_SCAN_1:
			for levelObj := 1; ; {
				switch r.ReadOne().ID {
				case gqlscan.TokenObj:
					levelObj++
				case gqlscan.TokenObjEnd:
					levelObj--
					if levelObj < 1 {
						break OBJ_SCAN_1
					}
				}
			}
			rAfter := *r
			*r = rBefore
			if fn(r) {
				return true
			}
			*r = rAfter
		case gqlscan.TokenArr:
			levelArr++
		VAL_SCAN:
			for levelArr > 0 {
				rBefore := *r
				switch r.ReadOne().ID {
				case gqlscan.TokenArrEnd:
					levelArr--
					if levelArr < 1 {
						break VAL_SCAN
					}
					goto VAL_SCAN
				case gqlscan.TokenArr:
					levelArr++
				case gqlscan.TokenObj:
				OBJ_SCAN_2:
					for levelObj := 1; ; {
						switch r.ReadOne().ID {
						case gqlscan.TokenObj:
							levelObj++
						case gqlscan.TokenObjEnd:
							levelObj--
							if levelObj < 1 {
								break OBJ_SCAN_2
							}
						}
					}
					rAfter := *r
					*r = rBefore
					if fn(r) {
						return true
					}
					*r = rAfter
				default:
					rAfter := *r
					*r = rBefore
					if fn(r) {
						return true
					}
					*r = rAfter
				}
			}
		}
	}
	return false
}
