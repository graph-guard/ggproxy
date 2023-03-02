package countval

import (
	"github.com/graph-guard/ggproxy/engine/playmon/internal/tokenreader"
	"github.com/graph-guard/gqlscan"
)

// Until counts the number of tokens and values
// until term is reached.
// Immediately returns true if any call to fn returned true as well.
func Until(r *tokenreader.Reader, term gqlscan.Token) (values, tokens int) {
	for ; !r.EOF() && r.PeekOne().ID != term; tokens++ {
		switch r.ReadOne().ID {
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
			values++
		case gqlscan.TokenArr:
			tokens++
		SCAN_ARR:
			for levelArr := 1; ; tokens++ {
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
			values++
		}
	}
	return values, tokens
}
