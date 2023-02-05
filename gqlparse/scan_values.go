package gqlparse

import "github.com/graph-guard/gqlscan"

// ScanValues calls fn for every value in a.
// Immediately returns true if any call to fn returned true as well.
func ScanValues(
	a []Token,
	fn func([]Token) (stop bool),
) (stopped bool) {
	for i := 0; i < len(a); i++ {
		switch a[i].ID {
		case gqlscan.TokenTrue,
			gqlscan.TokenFalse,
			gqlscan.TokenStr,
			gqlscan.TokenStrBlock,
			gqlscan.TokenFloat,
			gqlscan.TokenInt,
			gqlscan.TokenEnumVal,
			gqlscan.TokenNull:
			if fn(a[i : i+1]) {
				return true
			}
		case gqlscan.TokenObj:
			s := i
			i++
		SCAN_OBJ:
			for levelObj := 1; ; i++ {
				switch a[i].ID {
				case gqlscan.TokenObj:
					levelObj++
				case gqlscan.TokenObjEnd:
					levelObj--
					if levelObj < 1 {
						break SCAN_OBJ
					}
				}
			}
			if fn(a[s : i+1]) {
				return true
			}
		case gqlscan.TokenArr:
			s := i
			i++
		SCAN_ARR:
			for levelArr := 1; ; i++ {
				switch a[i].ID {
				case gqlscan.TokenArr:
					levelArr++
				case gqlscan.TokenArrEnd:
					levelArr--
					if levelArr < 1 {
						break SCAN_ARR
					}
				}
			}
			if fn(a[s : i+1]) {
				return true
			}
		}
	}
	return false
}
