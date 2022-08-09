// Package segmented provides a 2-dimensional indexed append-only array type.
package segmented

import (
	"github.com/graph-guard/gguard-proxy/utilities/container/hamap"
)

// Segment defines the start and end indexes of a segment.
type Segment struct{ Start, End int }

// Array is a 2D indexed append-only array
// where each segment is indexed by a key.
type Array[K hamap.KeyInterface, T any] struct {
	index        *hamap.Map[K, Segment]
	lastSegStart int
	data         []T
}

// New allocates a new instance of an indexed 2D append-only array.
func New[K hamap.KeyInterface, T any]() *Array[K, T] {
	return &Array[K, T]{
		index: hamap.New[K, Segment](1024, nil),
	}
}

// Len returns the number of stored segments.
func (i *Array[K, T]) Len() int {
	return i.index.Len()
}

// Reset removes all stored segments.
func (i *Array[K, T]) Reset() {
	i.index.Reset()
	i.lastSegStart, i.data = 0, i.data[:0]
}

// GetSegment returns the segment.
func (i *Array[K, T]) GetSegment(s Segment) []T {
	return i.data[s.Start:s.End]
}

// Append appends onto the last uncommited segment.
func (i *Array[K, T]) Append(t ...T) {
	i.data = append(i.data, t...)
}

// Cut commits the pending segment under key and
// returns the segment identifier.
// Returns Segment{Start: -1} if the key already exists.
func (i *Array[K, T]) Cut(key K) (s Segment) {
	i.index.SetFn(key, func(x *Segment) Segment {
		if x != nil {
			// Already exists
			s.Start = -1
			return Segment{}
		}
		// Add new
		s.Start = i.lastSegStart
		s.End = len(i.data)
		i.lastSegStart = s.End
		return s
	})
	return
}

// Get returns the segment by key.
// Returns nil if key doesn't exist.
func (i *Array[K, T]) Get(key K) []T {
	s, ok := i.index.Get(key)
	if !ok {
		return nil
	}
	return i.data[s.Start:s.End]
}

// VisitAll calls fn for every stored segment.
func (i *Array[K, T]) VisitAll(fn func(key K, s []T)) {
	i.index.VisitAll(func(k K, s Segment) {
		fn(k, i.data[s.Start:s.End])
	})
}
