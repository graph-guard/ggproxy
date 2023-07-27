package container_test

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/graph-guard/ggproxy/pkg/container"
)

func forEachImplB(
	b *testing.B,
	fn func(*testing.B, container.Mapper[[]byte, int]),
) {
	for _, impl := range implementations {
		b.Run(impl.Name, func(b *testing.B) {
			fn(b, impl.Make(0))
		})
	}
}

var (
	GI int
	GB bool
)

func BenchmarkAdd(b *testing.B) {
	for _, td := range []int{8, 64, 192, 512, 1024} {
		b.Run(fmt.Sprintf("%v", td), func(b *testing.B) {
			keys := MakeKeys(td)
			forEachImplB(b, func(b *testing.B, m container.Mapper[[]byte, int]) {
				for n := 0; n < b.N; n++ {
					m.Reset()
					for i := 0; i < len(keys); i++ {
						m.Set(keys[i], i)
					}
				}
			})
		})
	}
}

func BenchmarkSet(b *testing.B) {
	for _, td := range []int{8, 64, 192, 512, 1024} {
		b.Run(fmt.Sprintf("%v", td), func(b *testing.B) {
			forEachImplB(b, func(b *testing.B, m container.Mapper[[]byte, int]) {
				keys := SetNewKeys(td, m)
				b.ResetTimer()
				for n := 0; n < b.N; n++ {
					for i := 0; i < len(keys); i++ {
						m.Set(keys[i], n)
					}
				}
			})
		})
	}
}

func BenchmarkGet(b *testing.B) {
	for _, td := range []int{8, 64, 192, 512, 1024} {
		b.Run(fmt.Sprintf("%v", td), func(b *testing.B) {
			forEachImplB(b, func(b *testing.B, m container.Mapper[[]byte, int]) {
				keys := SetNewKeys(td, m)
				b.ResetTimer()
				for n, i := 0, -1; n < b.N; n++ {
					i++
					if i >= len(keys) {
						i = 0
					}
					GI, GB = m.Get(keys[i])
				}
			})
		})
	}
}

func MakeKeys(n int) [][]byte {
	keys := make([][]byte, n)
	for i := range keys {
		keys[i] = RandBytes(20)
	}
	return keys
}

func SetNewKeys(n int, m container.Mapper[[]byte, int]) [][]byte {
	keys := MakeKeys(n)
	for i := range keys {
		m.Set(keys[i], i)
	}
	return keys
}

func GetLast(m container.Mapper[[]byte, int]) (lastKey []byte) {
	i := 0
	m.Visit(func(key []byte, value int) bool {
		if i+1 == m.Len() {
			lastKey = key
		}
		i++
		return false
	})
	return nil
}

func RandBytes(n int) []byte {
	letters := []byte(
		"abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_",
	)
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return b
}
