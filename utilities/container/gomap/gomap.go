// package gomap provides a container.Mapper implementation
// backed by Go's native map for benchmark reference.
package gomap

type KeyInterface interface {
	string | []byte
}

type Gomap[K KeyInterface, V any] struct {
	m map[string]V
}

func New[K KeyInterface, V any](capacity int) *Gomap[K, V] {
	return &Gomap[K, V]{
		m: make(map[string]V, capacity),
	}
}

func (m *Gomap[K, V]) Set(key K, value V) {
	m.m[string(key)] = value
}

func (m *Gomap[K, V]) Delete(key K) {
	delete(m.m, string(key))
}

func (m *Gomap[K, V]) Get(key K) (v V, ok bool) {
	v, ok = m.m[string(key)]
	return v, ok
}

func (m *Gomap[K, V]) Reset() {
	m.m = make(map[string]V)
}

func (m *Gomap[K, V]) Len() int {
	return len(m.m)
}

func (m *Gomap[K, V]) Visit(fn func(K, V) bool) {
	for k, v := range m.m {
		if !fn(K(k), v) {
			break
		}
	}
}
