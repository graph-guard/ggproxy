package mhstore_test

import (
	"encoding/binary"
	"math/rand"
	"testing"

	"github.com/graph-guard/ggproxy/utilities/mhstore"
	"github.com/stretchr/testify/require"
)

func TestReset(t *testing.T) {
	store := mhstore.New()
	store.Add(0, RandUint64())
	store.Reset()
	require.Equal(t, 0, store.Len())
}

func TestAdd(t *testing.T) {
	store := mhstore.New()
	store.Add(0, 42)
	store.Add(0, 0)
	store.Add(64, 64)
	require.Equal(t, 2, store.Len())
	require.Equal(t, []uint64{0, 42}, store.Get(0))
	require.Equal(t, []uint64{64}, store.Get(64))
}

func TestLen(t *testing.T) {
	masks := map[uint16]bool{}
	for len(masks) < 384 {
		masks[RandUint16()] = true
	}

	store := mhstore.New()
	for m := range masks {
		for i := 0; i < 384; i++ {
			store.Add(m, RandUint64())
		}
	}

	require.Equal(t, 384, store.Len())
}

func RandUint64() uint64 {
	buf := make([]byte, 8)
	rand.Read(buf)

	return binary.LittleEndian.Uint64(buf)
}

func RandUint16() uint16 {
	buf := make([]byte, 2)
	rand.Read(buf)

	return binary.LittleEndian.Uint16(buf)
}
