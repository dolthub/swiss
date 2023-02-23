//go:build ignore
// +build ignore

package main

func main() {
	TEXT("MatchMetadata", NOSPLIT, "func(metadata *[16]int8, hash int8) uint16")
	Doc("MatchMetadata matches |hash| against each byte in |metadata|.")
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
