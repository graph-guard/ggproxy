package gqlparse

import "github.com/graph-guard/gqlscan"

// CountValuesUntil counts the number of tokens and values
// until term is reached.
// Immediately returns true if any call to fn returned true as well.
func CountValuesUntil(a []Token, term gqlscan.Token) (values, tokens int) {
	for ; tokens < len(a) && a[tokens].ID != term; tokens++ {
		switch a[tokens].ID {
		case gqlscan.TokenTrue,
			gqlscan.TokenFalse,
			gqlscan.TokenStr,
			gqlscan.TokenStrBlock,
			gqlscan.TokenFloat,
			gqlscan.TokenInt,
			gqlscan.TokenEnumVal,
			gqlscan.TokenNull:
			values++
		case gqlscan.TokenObj:
			tokens++
		SCAN_OBJ:
			for levelObj := 1; ; tokens++ {
				switch a[tokens].ID {
				case gqlscan.TokenObj:
					levelObj++
				case gqlscan.TokenObjEnd:
					levelObj--
					if levelObj < 1 {
						break SCAN_OBJ
					}
				}
			}
			values++
		case gqlscan.TokenArr:
			tokens++
		SCAN_ARR:
			for levelArr := 1; ; tokens++ {
				switch a[tokens].ID {
				case gqlscan.TokenArr:
					levelArr++
				case gqlscan.TokenArrEnd:
					levelArr--
					if levelArr < 1 {
						break SCAN_ARR
					}
				}
			}
			values++
		}
	}
	return values, tokens
}
