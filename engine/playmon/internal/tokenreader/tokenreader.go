// Package tokenreader provides Reader which reads a continuous
// stream of tokens respecting variable index tokens.
package tokenreader

import (
	"github.com/graph-guard/ggproxy/gqlparse"
	"github.com/graph-guard/gqlscan"
)

// Reader reads a continuous stream of tokens respecting
// variable index tokens.
type Reader struct {
	Vars      [][]gqlparse.Token
	Var, Main []gqlparse.Token
}

// ReadOne advances the reader by one token and returns it.
func (r *Reader) ReadOne() (t gqlparse.Token) {
	if len(r.Var) > 0 {
		t, r.Var = r.Var[0], r.Var[1:]
		return t
	}
	if vi := r.Main[0].VariableIndex(); vi > -1 {
		t, r.Var, r.Main = r.Vars[vi][0], r.Vars[vi][1:], r.Main[1:]
		return t
	}
	t, r.Main = r.Main[0], r.Main[1:]
	return t
}

// SkipUntil skips all tokens until (including) the first token
// that has the given id.
func (r *Reader) SkipUntil(id gqlscan.Token) {
	for r.ReadOne().ID != id {
	}
}

// EOF returns true if the reader is currently at the end of the file,
// otherwise returns false.
func (r *Reader) EOF() bool {
	return len(r.Var) < 1 && len(r.Main) < 1
}

// PeekOne returns true if the reader is currently at the end of the file,
// otherwise returns false.
func (r *Reader) PeekOne() gqlparse.Token {
	if len(r.Var) > 0 {
		return r.Var[0]
	}
	if vi := r.Main[0].VariableIndex(); vi > -1 {
		r.Var, r.Main = r.Vars[vi], r.Main[1:]
		return r.Var[0]
	}
	return r.Main[0]
}
