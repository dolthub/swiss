//go:build !amd64

package simd

func MatchMetadata(metadata *[16]int8, hash int8) (b uint16) {
	return matchMetadata(metadata, hash)
}
