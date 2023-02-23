package simd

import "unsafe"

func matchMetadata(metadata *[16]int8, hash int8) (b uint16) {
	b |= uint16(castBool(metadata[0] == hash)) << 0
	b |= uint16(castBool(metadata[1] == hash)) << 1
	b |= uint16(castBool(metadata[2] == hash)) << 2
	b |= uint16(castBool(metadata[3] == hash)) << 3
	b |= uint16(castBool(metadata[4] == hash)) << 4
	b |= uint16(castBool(metadata[5] == hash)) << 5
	b |= uint16(castBool(metadata[6] == hash)) << 6
	b |= uint16(castBool(metadata[7] == hash)) << 7
	b |= uint16(castBool(metadata[8] == hash)) << 8
	b |= uint16(castBool(metadata[9] == hash)) << 9
	b |= uint16(castBool(metadata[10] == hash)) << 10
	b |= uint16(castBool(metadata[11] == hash)) << 11
	b |= uint16(castBool(metadata[12] == hash)) << 12
	b |= uint16(castBool(metadata[13] == hash)) << 13
	b |= uint16(castBool(metadata[14] == hash)) << 14
	b |= uint16(castBool(metadata[15] == hash)) << 15
	return
}

func castBool(b bool) int8 {
	return *(*int8)((unsafe.Pointer)(&b))
}
