package swiss

import (
	"math/bits"

	"github.com/dolthub/swiss/simd"
)

const (
	h1Mask uint64 = 0xffff_ffff_ffff_ff80
	h2Mask uint64 = 0x0000_0000_0000_007f

	empty     int8 = -128 // 0b1000_0000
	tombstone int8 = -2   // 0b1111_1110
)

// h1 is a 57 bit hash prefix
type h1 uint64

// h2 is a 7 bit hash suffix
type h2 int8

func splitHash(h uint64) (h1, h2) {
	return h1((h & h1Mask) >> 7), h2(h & h2Mask)
}

// metadata is the h2 metadata array for a group.
// find operations first probe the controls bytes
// to filter candidates before matching keys
type metadata [16]int8

type bitset uint16

func newEmptyMetadata() metadata {
	return metadata{
		empty, empty, empty, empty,
		empty, empty, empty, empty,
		empty, empty, empty, empty,
		empty, empty, empty, empty,
	}
}

func metaMatchH2(m *metadata, h h2) bitset {
	b := simd.MatchMetadata((*[16]int8)(m), int8(h))
	return bitset(b)
}

func metaMatchEmpty(m *metadata) bitset {
	b := simd.MatchMetadata((*[16]int8)(m), empty)
	return bitset(b)
}

func nextPow2(x uint32) uint32 {
	return 1 << (32 - bits.LeadingZeros32(x-1))
}

// lemire.me/blog/2016/06/27/a-fast-alternative-to-the-modulo-reduction/
func fastModN(x, n uint32) uint32 {
	return uint32((uint64(x) * uint64(n)) >> 32)
}
