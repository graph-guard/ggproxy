package xxhash_test

import (
	"fmt"
	"testing"

	"github.com/graph-guard/ggproxy/pkg/xxhash"

	"github.com/pierrec/xxHash/xxHash64"
	"github.com/stretchr/testify/require"
)

// TestWriteAndSum makes sure the original and forked code
// produce the same results.
func TestWriteAndSum(t *testing.T) {
	for _, seed := range []uint64{
		0, 1, 5134, 2598712366, 936583347421323,
	} {
		t.Run(fmt.Sprintf("%d", seed), func(t *testing.T) {
			in := []string{"foo", "bar"}

			t.Run("Write", func(t *testing.T) {
				require := require.New(t)
				h, oh := xxhash.New(seed), xxHash64.New(seed)
				for _, in := range in {
					xxhash.Write(&h, in)
					n, err := oh.Write([]byte(in))
					require.Equal(len([]byte(in)), n)
					require.NoError(err)
					require.Equal(oh.Sum64(), h.Sum64())
				}
			})

			t.Run("Write8", func(t *testing.T) {
				require := require.New(t)
				h, oh := xxhash.New(seed), xxHash64.New(seed)
				bytes := [8]byte{1, 2, 3, 4, 5, 6, 7, 8}
				for _, in := range in {
					xxhash.Write(&h, in)
					xxhash.Write8(&h, bytes)
					n, err := oh.Write([]byte(in))
					require.Equal(len([]byte(in)), n)
					require.NoError(err)
					n, err = oh.Write(bytes[:])
					require.Equal(8, n)
					require.NoError(err)
					require.Equal(oh.Sum64(), h.Sum64())
				}
			})
		})
	}
}
