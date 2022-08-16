package amap

import (
	"github.com/graph-guard/ggproxy/utilities/math"
)

type KeyInterface interface {
	uint8 | uint16 | uint32 | uint64 | int | int8 | int16 | int32 | int64 | float32 | float64
}

// Entity is a key-value pair.
type Entity[K KeyInterface, V any] struct {
	Key   K
	Value V
}

// Map is a map implementation for strings based on arrays and binary search.
// Perfect for small to medium data amounts. Resetable.
type Map[K KeyInterface, V any] struct {
	A []Entity[K, V]
}

// New creates a new instance of Map.
func New[K KeyInterface, V any](capacity int, entities ...Entity[K, V]) *Map[K, V] {
	am := &Map[K, V]{
		A: make([]Entity[K, V], 0, capacity),
	}
	for _, e := range entities {
		am.Set(e.Key, e.Value)
	}

	return am
}

// Reset resets the map.
func (am *Map[K, V]) Reset() {
	am.A = am.A[:0]
}

// Set associates key with value overwriting any existing associations.
func (am *Map[K, V]) Set(key K, value V) (idx int) {
	var found bool
	if len(am.A) >= 256 {
		idx, found = findExp(am.A, key)
	} else {
		idx, found = findBin(am.A, key, 0, len(am.A)-1)
	}
	if found {
		am.A[idx].Value = value
		return
	}

	if idx == len(am.A) {
		am.A = append(am.A, Entity[K, V]{key, value})
	} else {
		am.A = append(am.A[:idx+1], am.A[idx:]...)
		am.A[idx] = Entity[K, V]{key, value}
	}

	return
}

// SetFn associates key with value or applies function to existing association.
func (am *Map[K, V]) SetFn(key K, value V, fn func(value *V)) {
	var idx int
	var found bool
	if len(am.A) >= 256 {
		idx, found = findExp(am.A, key)
	} else {
		idx, found = findBin(am.A, key, 0, len(am.A)-1)
	}
	if found {
		fn(&am.A[idx].Value)
		return
	}

	if idx == len(am.A) {
		am.A = append(am.A, Entity[K, V]{key, value})
	} else {
		am.A = append(am.A[:idx+1], am.A[idx:]...)
		am.A[idx] = Entity[K, V]{key, value}
	}
}

// Get returns (value, true) if key exists,
// otherwise returns (zeroValue, false).
func (am *Map[K, V]) Get(key K) (value V, ok bool) {
	var idx int
	var found bool
	if len(am.A) >= 256 {
		idx, found = findExp(am.A, key)
	} else {
		idx, found = findBin(am.A, key, 0, len(am.A)-1)
	}

	if found {
		return am.A[idx].Value, true
	}
	return value, false
}

// Delete removes the element at the given index.
// Noop if the index is out of bound.
func (am *Map[K, V]) Delete(key K) {
	if i := am.Index(key); i > -1 {
		am.A = append(am.A[:i], am.A[i+1:]...)
	}
}

// DeleteByIndex removes the element at the given index.
// Noop if the index is out of bound.
func (am *Map[K, V]) DeleteByIndex(index int) (key K, value V) {
	if index < 0 || index >= len(am.A) {
		return
	}
	key, value = am.A[index].Key, am.A[index].Value
	am.A = append(am.A[:index], am.A[index+1:]...)
	return
}

// Index returns the index of the entry or -1 if it wasn't found.
func (am *Map[K, V]) Index(key K) int {
	var idx int
	var found bool
	if len(am.A) >= 256 {
		idx, found = findExp(am.A, key)
	} else {
		idx, found = findBin(am.A, key, 0, len(am.A)-1)
	}

	if found {
		return idx
	}

	return -1
}

// Len returns the number of stored key-value pairs.
func (am *Map[K, V]) Len() int {
	return len(am.A)
}

// Visit calls fn for every stored key-value pair.
// Returns immediately if fn returns true.
func (am *Map[K, V]) Visit(fn func(key K, value V) (stop bool)) {
	for i := range am.A {
		if fn(am.A[i].Key, am.A[i].Value) {
			break
		}
	}
}

// findExp utilizes exponential binary search and returns index and true if
// the element was found, otherwise returns bound and false.
func findExp[K KeyInterface, V any](
	e []Entity[K, V],
	key K,
) (int, bool) {
	l, r := 0, 1

	if len(e) > 1 {
		for r < len(e) && e[r].Key < key {
			l = r
			r = r << 1
		}
	}

	return findBin(e, key, l, math.Min(r, len(e)-1))
}

// findBin utilizes binary search and returns index and true if
// the element was found, otherwise returns left bound and false.
func findBin[K KeyInterface, V any](
	e []Entity[K, V],
	key K,
	l, r int,
) (int, bool) {
	for l <= r {
		m := l + (r-l)>>1

		if e[m].Key == key {
			return m, true
		}

		if e[m].Key > key {
			r = m - 1
		} else {
			l = m + 1
		}
	}

	return l, false
}
