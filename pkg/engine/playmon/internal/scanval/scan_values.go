package scanval

import (
	"github.com/graph-guard/ggproxy/pkg/engine/playmon/internal/tokenreader"
	"github.com/graph-guard/gqlscan"
)

// ScanValues calls fn for every value in a.
// Immediately returns true if any call to fn returned true as well.
func ScanValues(
	r *tokenreader.Reader,
	fn func(r *tokenreader.Reader) (stop bool),
) (stopped bool) {
	for !r.EOF() {
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
		SCAN_OBJ:
			for levelObj := 1; ; {
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
			rAfter := *r
			*r = rBefore
			if fn(r) {
				return true
			}
			*r = rAfter
		case gqlscan.TokenArr:
		SCAN_ARR:
			for levelArr := 1; ; {
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
			rAfter := *r
			*r = rBefore
			if fn(r) {
				return true
			}
			*r = rAfter
		}
	}
	return false
}
