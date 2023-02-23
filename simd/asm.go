//go:build ignore
// +build ignore

package main

import (
	. "github.com/mmcloughlin/avo/build"
	. "github.com/mmcloughlin/avo/operand"
)

func main() {
	ConstraintExpr("amd64")

	TEXT("MatchMetadata", NOSPLIT, "func(metadata *[16]int8, hash int8) uint16")
	Doc("MatchMetadata performs a 16-way probe of |metadata| using SSE instructions",
		"nb: |metadata| must be an aligned pointer")
	m := Mem{Base: Load(Param("metadata"), GP64())}
	h := Load(Param("hash"), GP64())

	meta := XMM()
	MOVUPS(m, meta)
	matches := XMM()
	MOVDQ2Q(h, matches)

	mask := GP32()
	VPBROADCASTB(matches, matches)
	PCMPEQB(meta, matches)
	PMOVMSKB(matches, mask)

	Store(mask.As16(), ReturnIndex(0))
	RET()
	Generate()
}
