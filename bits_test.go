package swiss

import (
	"math/rand"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"
)

func TestNextPow2(t *testing.T) {
	assert.Equal(t, 0, int(nextPow2(0)))
	assert.Equal(t, 1, int(nextPow2(1)))
	assert.Equal(t, 2, int(nextPow2(2)))
	assert.Equal(t, 4, int(nextPow2(3)))
	assert.Equal(t, 8, int(nextPow2(7)))
	assert.Equal(t, 8, int(nextPow2(8)))
	assert.Equal(t, 16, int(nextPow2(9)))
}

func TestConstants(t *testing.T) {
	c1, c2 := empty, tombstone
	assert.Equal(t, byte(0b1000_0000), byte(c1))
	assert.Equal(t, byte(0b1000_0000), reinterpretCast(c1))
	assert.Equal(t, byte(0b1111_1110), byte(c2))
	assert.Equal(t, byte(0b1111_1110), reinterpretCast(c2))
}

func reinterpretCast(i int8) byte {
	return *(*byte)(unsafe.Pointer(&i))
}

func TestFastMod(t *testing.T) {
	t.Run("n=10", func(t *testing.T) {
		testFastMod(t, 10)
	})
	t.Run("n=100", func(t *testing.T) {
		testFastMod(t, 100)
	})
	t.Run("n=1000", func(t *testing.T) {
		testFastMod(t, 1000)
	})
}

func testFastMod(t *testing.T, n uint32) {
	const trials = 32 * 1024
	for i := 0; i < trials; i++ {
		x := rand.Uint32()
		y := fastModN(x, n)
		assert.Less(t, y, n)
		t.Logf("fastMod(%d, %d): %d", x, n, y)
	}
}
