// Package decl provides a helper for getting the file:line declaration
package decl

import (
	"fmt"
	"path/filepath"
	"runtime"
)

func New[T any](data T) Declaration[T] {
	return Declaration[T]{
		Decl: decl(2),
		Data: data,
	}
}

type Declaration[T any] struct {
	Decl string
	Data T
}

func decl(skipFrames int) string {
	_, filename, line, _ := runtime.Caller(skipFrames)
	return fmt.Sprintf("%s:%d", filepath.Base(filename), line)
}
