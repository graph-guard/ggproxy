package container

type KeyInterface interface {
	string | []byte
}

type Mapper[K KeyInterface, V any] interface {
	Set(K, V)
	Get(K) (v V, ok bool)
	Reset()
	Len() int
	Delete(K)
	Visit(func(K, V) bool)
}
