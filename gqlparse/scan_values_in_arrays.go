package gqlparse

import "github.com/graph-guard/gqlscan"

// ScanValuesInArrays calls fn for every non-array value contained in x recursively.
// Immediately returns true if any call to fn returned true as well.
func ScanValuesInArrays(
	x []Token,
	fn func([]Token) (stop bool),
) (stopped bool) {
	for levelArr, i := 0, 0; i < len(x); i++ {
		switch x[i].ID {
		case gqlscan.TokenTrue,
			gqlscan.TokenFalse,
			gqlscan.TokenStr,
			gqlscan.TokenStrBlock,
			gqlscan.TokenFloat,
			gqlscan.TokenInt,
			gqlscan.TokenEnumVal,
			gqlscan.TokenNull:
			if fn(x[i : i+1]) {
				return true
			}
		case gqlscan.TokenObj:
			s := i
			i++
		OBJ_SCAN_1:
			for levelObj := 1; ; i++ {
				switch x[i].ID {
				case gqlscan.TokenObj:
					levelObj++
				case gqlscan.TokenObjEnd:
					levelObj--
					if levelObj < 1 {
						i++
						break OBJ_SCAN_1
					}
				}
			}
			if fn(x[s:i]) {
				return true
			}
		case gqlscan.TokenArr:
			levelArr++
			i++
		VAL_SCAN:
			for levelArr > 0 {
				switch x[i].ID {
				case gqlscan.TokenArrEnd:
					i++
					levelArr--
					if levelArr < 1 {
						break VAL_SCAN
					}
					goto VAL_SCAN
				case gqlscan.TokenArr:
					i++
					levelArr++
				case gqlscan.TokenObj:
					s := i
					i++
				OBJ_SCAN_2:
					for levelObj := 1; ; i++ {
						switch x[i].ID {
						case gqlscan.TokenObj:
							levelObj++
						case gqlscan.TokenObjEnd:
							levelObj--
							if levelObj < 1 {
								i++
								break OBJ_SCAN_2
							}
						}
					}
					if fn(x[s:i]) {
						return true
					}
				default:
					if fn(x[i : i+1]) {
						return true
					}
					i++
				}
			}
		}
	}
	return false
}
