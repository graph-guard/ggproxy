// package linear provides a container.Mapper implementation
// backed by a slice and linear search for benchmark reference.
package linear

type KeyInterface interface {
	string | []byte
}

type bucket[K KeyInterface, V any] struct {
	Key   K
	Value V
}

type Linear[K KeyInterface, V any] struct {
	d []bucket[K, V]
}

func New[K KeyInterface, V any](capacity int) *Linear[K, V] {
	return &Linear[K, V]{
		d: make([]bucket[K, V], 0, capacity),
	}
}

func (m *Linear[K, V]) Set(key K, value V) {
	for i := 0; i < len(m.d); i++ {
		if string(m.d[i].Key) == string(key) {
			m.d[i].Value = value
			return
		}
	}
	m.d = append(m.d, bucket[K, V]{
		Key:   key,
		Value: value,
	})
}

func (m *Linear[K, V]) Delete(key K) {
	for i := 0; i < len(m.d); i++ {
		if string(m.d[i].Key) == string(key) {
			m.d[i] = m.d[len(m.d)-1]
			m.d = m.d[:len(m.d)-1]
			return
		}
	}
}

func (m *Linear[K, V]) Get(key K) (v V, ok bool) {
	for i := 0; i < len(m.d); i++ {
		if string(m.d[i].Key) == string(key) {
			return m.d[i].Value, true
		}
	}
	return v, false
}

func (m *Linear[K, V]) Reset() {
	m.d = m.d[:0]
}

func (m *Linear[K, V]) Len() int {
	return len(m.d)
}

func (m *Linear[K, V]) Visit(fn func(K, V) bool) {
	for i := 0; i < len(m.d); i++ {
		if !fn(m.d[i].Key, m.d[i].Value) {
			break
		}
	}
}
