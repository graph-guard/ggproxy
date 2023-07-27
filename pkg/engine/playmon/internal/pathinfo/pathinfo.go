package pathinfo

import "github.com/graph-guard/gqt/v4"

// Info returns the number of `max` sets expression e is inside
// and the most relevant (parent) `max` set, if any.
func Info(e gqt.Expression) (depth int, parent *gqt.SelectionMax) {
	for ; e != nil; e = e.GetParent() {
		if s, ok := e.(*gqt.SelectionMax); ok {
			if parent == nil {
				parent = s
			}
			depth++
		}
	}
	return depth, parent
}
