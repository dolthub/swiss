//go:build amd64 && !nosimd

package swiss

import (
	"math/bits"

	"github.com/dolthub/swiss/simd"
)

const (
	groupSize       = 16
	maxAvgGroupLoad = 14
)

type bitset uint16

func metaMatchH2(m *metadata, h h2) bitset {
	b := simd.MatchMetadata((*[16]int8)(m), int8(h))
	return bitset(b)
}

func metaMatchEmpty(m *metadata) bitset {
	b := simd.MatchMetadata((*[16]int8)(m), empty)
	return bitset(b)
}

func nextMatch(b *bitset) (s uint32) {
	s = uint32(bits.TrailingZeros16(uint16(*b)))
	*b &= ^(1 << s) // clear bit |s|
	return
}
