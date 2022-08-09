package aset

import "github.com/graph-guard/gguard-proxy/utilities/math"

type ElementInterface interface {
	uint8 | uint16 | uint32 | uint64 | int | int8 | int16 | int32 | int64 | float32 | float64
}

// Set is a sorted set implementation for numbers based on arrays and binary search.
type Set[T ElementInterface] struct {
	a []T
}

// New creates a new instance of Set.
func New[T ElementInterface](capacity int, elements ...T) *Set[T] {
	s := &Set[T]{
		a: make([]T, 0, capacity),
	}
	for _, el := range elements {
		s.Add(el)
	}

	return s
}

// Reset resets the set.
func (as *Set[T]) Reset() {
	as.a = as.a[:0]
}

// Add adds a new element to the set.
func (as *Set[T]) Add(el T) {
	var idx int
	var found bool
	if len(as.a) >= 256 {
		idx, found = FindExp(as.a, el)
	} else {
		idx, found = FindBin(as.a, el, 0, len(as.a)-1)
	}
	if !found {
		if idx == len(as.a) {
			as.a = append(as.a, el)
		} else {
			as.a = append(as.a[:idx+1], as.a[idx:]...)
			as.a[idx] = el
		}
	}
}

// Get returns an element at the index.
func (as *Set[T]) Get(idx int) T {
	return as.a[idx]
}

// Delete returns and removes an element at the index.
func (as *Set[T]) Delete(idx int) T {
	el := as.a[idx]
	as.a = append(as.a[:idx], as.a[idx+1:]...)

	return el
}

// Find searches for an element and returns it if found or -1 if not found.
func (as *Set[T]) Find(el T) int {
	var idx int
	var found bool
	if len(as.a) >= 256 {
		idx, found = FindExp(as.a, el)
	} else {
		idx, found = FindBin(as.a, el, 0, len(as.a)-1)
	}

	if found {
		return idx
	}

	return -1
}

// Len returns the set length.
func (as *Set[T]) Len() int {
	return len(as.a)
}

// Visit loops through the set. Breaks if true is returned by the fn function.
func (as *Set[T]) Visit(fn func(T) (stop bool)) {
	for i := range as.a {
		if fn(as.a[i]) {
			break
		}
	}
}

// FindExp is an exponential binary search imlementation.
// Returns either index and true if the element found or left bound and false if not found.
func FindExp[T ElementInterface](s []T, el T) (int, bool) {
	l := 0
	r := 1

	if len(s) > 1 {
		for r < len(s) && s[r] < el {
			l = r
			r = r << 1
		}
	}

	return FindBin(s, el, l, math.Min(r, len(s)-1))
}

// FindBin is an binary search implementation.
// Returns either index and true if the element found or left bound and false if not found.
func FindBin[T ElementInterface](s []T, el T, l, r int) (int, bool) {
	for l <= r {
		m := l + (r-l)>>1

		if s[m] == el {
			return m, true
		}

		if s[m] > el {
			r = m - 1
		} else {
			l = m + 1
		}
	}

	return l, false
}
