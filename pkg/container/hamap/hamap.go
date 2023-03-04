// package hamap provides a collision-safe hashmap implementation
// which is more efficient than Go's native map for small datasets
// and allows allocation-free reseting efficiently reusing memory.
// Allocations are made only in case of rare hash collisions.
// Any custom hasher can be provided during initialization.
// By default, XXH3 from github.com/zeebo/xxh3 is used with seed 0.
// On an Apple M1 Max machine the efficiency breaking point
// compared to Go's native map is at around 192 items.
package hamap

import (
	"github.com/google/go-cmp/cmp"
	"github.com/graph-guard/ggproxy/pkg/math"
	"github.com/zeebo/xxh3"
)

type KeyInterface interface{ string | []byte }

type bucket[K KeyInterface, V any] struct {
	KeyHash uint64
	pair[K, V]
}

type pair[K KeyInterface, V any] struct {
	Key   K
	Value V
	Next  *pair[K, V]
}

type Hasher[K KeyInterface] interface{ Hash(K) uint64 }

// Map is backed by a slice and utilizes binary search.
//
// WARNING: In case of []byte typed keys the keys will
// be aliased and must remain immutable until the map is reset!
type Map[K KeyInterface, V any] struct {
	size   int
	d      []bucket[K, V]
	hasher Hasher[K]
}

func (m *Map[K, V]) Equal(mm *Map[K, V]) bool {
	return m.size == mm.size && cmp.Equal(m.d, mm.d) && m.hasher == mm.hasher
}

// HasherXXH3 can be used to provide custom seeds during initialization.
type HasherXXH3[K KeyInterface] struct {
	Seed uint64
}

// Hash hashes k to a 64-bit hash value.
func (h *HasherXXH3[K]) Hash(k K) uint64 {
	return xxh3.HashSeed([]byte(k), h.Seed)
}

var (
	defaultHasherS = &HasherXXH3[string]{}
	defaultHasherB = &HasherXXH3[[]byte]{}
)

// New creates a new map instance.
func New[K KeyInterface, V any](
	capacity int,
	hasher Hasher[K],
) *Map[K, V] {
	if hasher == nil {
		var zeroKey K
		switch any(zeroKey).(type) {
		case string:
			hasher = (*HasherXXH3[K])(defaultHasherS)
		case []byte:
			hasher = (*HasherXXH3[K])(defaultHasherB)
		}
	}
	return &Map[K, V]{
		d:      make([]bucket[K, V], 0, capacity),
		hasher: hasher,
	}
}

// Pair is a key-value pair.
type Pair[K KeyInterface, V any] struct {
	Key   K
	Value V
}

// Reset resets the map
func (m *Map[K, V]) Reset() {
	m.d, m.size = m.d[:0], 0
}

// Set associates key with value overwriting any existing associations.
//
// WARNING: In case of []byte typed keys the map will alias keys!
// Make sure key remains immutable during the life-time of the map
// or until the map is reset.
func (m *Map[K, V]) Set(key K, value V) {
	hash := m.hasher.Hash(key)
	i, found := m.index(hash)
	if found {
		for p := &m.d[i].pair; ; p = p.Next {
			if string(p.Key) == string(key) {
				p.Value = value
				return
			}
			if p.Next == nil {
				// Key doesn't yet exist
				m.size++
				p.Next = &pair[K, V]{Key: key, Value: value}
				return
			}
		}
	}

	m.size++
	if i == len(m.d) {
		m.d = append(m.d, bucket[K, V]{
			hash, pair[K, V]{Key: key, Value: value},
		})
		return
	}
	m.d = append(m.d[:i+1], m.d[i:]...)
	m.d[i] = bucket[K, V]{
		hash, pair[K, V]{Key: key, Value: value},
	}
}

// SetFn calls fn(nil) if the key doesn't exist yet and associates
// the value returned by fn with the key. If the key already exists
// then fn is passed a pointer to the value already associated with the key.
//
// WARNING: In case of []byte typed keys the map will alias keys!
// Make sure key remains immutable during the life-time of the map
// or until the map is reset.
func (m *Map[K, V]) SetFn(key K, fn func(*V) V) {
	hash := m.hasher.Hash(key)
	i, found := m.index(hash)
	if found {
		for p := &m.d[i].pair; ; p = p.Next {
			if string(p.Key) == string(key) {
				_ = fn(&p.Value)
				return
			}
			if p.Next == nil {
				// Key doesn't yet exist
				m.size++
				p.Next = &pair[K, V]{Key: key, Value: fn(nil)}
				return
			}
		}
	}

	m.size++
	if i == len(m.d) {
		m.d = append(m.d, bucket[K, V]{
			hash, pair[K, V]{Key: key, Value: fn(nil)},
		})
	} else {
		m.d = append(m.d[:i+1], m.d[i:]...)
		m.d[i] = bucket[K, V]{
			hash, pair[K, V]{Key: key, Value: fn(nil)},
		}
	}
}

// Get returns (value, true) if key exists,
// otherwise returns (zeroValue, false).
func (m *Map[K, V]) Get(key K) (value V, ok bool) {
	hash := m.hasher.Hash(key)
	if i, found := m.index(hash); found {
		if m.d[i].Next == nil {
			return m.d[i].Value, true
		}
		for p := &m.d[i].pair; p != nil; p = p.Next {
			if string(p.Key) == string(key) {
				return p.Value, true
			}
		}
	}
	return value, false
}

// GetFn calls fn providing a pointer to the value and
// returns true if key exists,
// otherwise calls fn providing nil and returns false.
func (m *Map[K, V]) GetFn(key K, fn func(*V)) (ok bool) {
	hash := m.hasher.Hash(key)
	if i, found := m.index(hash); found {
		if m.d[i].Next == nil {
			fn(&m.d[i].Value)
			return true
		}
		for p := &m.d[i].pair; p != nil; p = p.Next {
			if string(p.Key) == string(key) {
				fn(&p.Value)
				return true
			}
		}
	}
	return false
}

func (m *Map[K, V]) index(keyHash uint64) (i int, found bool) {
	if len(m.d) >= 256 {
		return findExp(m.d, keyHash)
	}
	return findBin(m.d, keyHash, 0, len(m.d)-1)
}

// Delete deletes the key if it exists.
// Noop if the key doesn't exist.
func (m *Map[K, V]) Delete(key K) {
	hash := m.hasher.Hash(key)
	if i, found := m.index(hash); found {
		if m.d[i].Next == nil && string(key) == string(m.d[i].Key) {
			m.d = append(m.d[:i], m.d[i+1:]...)
			m.size--
			return
		}

		// Hash collision
		var prev *pair[K, V]
		for p := &m.d[i].pair; ; {
			if string(p.Key) == string(key) {
				if prev == nil {
					// No parent
					if p.Next != nil {
						m.d[i].pair = *p.Next
					}
				} else {
					// Has parent
					prev.Next = p.Next
				}
				m.size--
				return
			}
			if p.Next == nil {
				return
			}
			prev, p = p, p.Next
		}
	}
}

// Len returns the number of stored key-value pairs.
func (m *Map[K, V]) Len() int {
	return m.size
}

// Visit calls fn for every stored key-value pair.
// Returns immediately if fn returns true.
func (m *Map[K, V]) Visit(fn func(key K, value V) (stop bool)) {
	for i := range m.d {
		if m.d[i].Next != nil {
			// Traverse linked list
			for p := &m.d[i].pair; p != nil; p = p.Next {
				if fn(p.Key, p.Value) {
					break
				}
			}
			continue
		}
		if fn(m.d[i].Key, m.d[i].Value) {
			break
		}
	}
}

// VisitAll calls fn for every stored key-value pair.
func (m *Map[K, V]) VisitAll(fn func(key K, value V)) {
	for i := range m.d {
		if m.d[i].Next != nil {
			// Traverse linked list
			for p := &m.d[i].pair; p != nil; p = p.Next {
				fn(p.Key, p.Value)
			}
			continue
		}
		fn(m.d[i].Key, m.d[i].Value)
	}
}

// Values returns all map values
func (m *Map[K, V]) Values() (values []V) {
	m.VisitAll(func(key K, value V) {
		values = append(values, value)
	})

	return
}

// findExp utilizes exponential binary search and returns index and true if
// the element was found, otherwise returns bound and false.
func findExp[K KeyInterface, V any](
	e []bucket[K, V],
	keyHash uint64,
) (int, bool) {
	l, r := 0, 1

	if len(e) > 1 {
		for r < len(e) && e[r].KeyHash < keyHash {
			l = r
			r = r << 1
		}
	}

	return findBin(e, keyHash, l, math.Min(r, len(e)-1))
}

// findBin utilizes binary search and returns index and true if
// the element was found, otherwise returns left bound and false.
func findBin[K KeyInterface, V any](
	e []bucket[K, V],
	keyHash uint64,
	l, r int,
) (int, bool) {
	for l <= r {
		m := l + (r-l)>>1

		if e[m].KeyHash == keyHash {
			return m, true
		}

		if e[m].KeyHash > keyHash {
			r = m - 1
		} else {
			l = m + 1
		}
	}

	return l, false
}
