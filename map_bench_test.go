package swiss

import (
	"math/bits"
	"math/rand"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/thepudds/swisstable"

	"github.com/stretchr/testify/assert"
)

func BenchmarkStringMaps(b *testing.B) {
	const keySz = 8
	sizes := []int{16, 128, 1024, 8192, 131072}
	for _, n := range sizes {
		b.Run("n="+strconv.Itoa(n), func(b *testing.B) {
			b.Run("runtime map", func(b *testing.B) {
				benchmarkRuntimeMap(b, genStringData(keySz, n))
			})
			b.Run("swiss.Map", func(b *testing.B) {
				benchmarkSwissMap(b, genStringData(keySz, n))
			})
		})
	}
}

func BenchmarkInt64Maps(b *testing.B) {
	sizes := []int{16, 128, 1024, 8192, 131072}
	for _, n := range sizes {
		b.Run("n="+strconv.Itoa(n), func(b *testing.B) {
			b.Run("runtime map", func(b *testing.B) {
				benchmarkRuntimeMap(b, generateInt64Data(n))
			})
			b.Run("swiss.Map", func(b *testing.B) {
				benchmarkSwissMap(b, generateInt64Data(n))
			})
			b.Run("swisstable.Map", func(b *testing.B) {
				benchmarkThepuddsMap(b, generateInt64Data(n))
			})
		})
	}
}

func BenchmarkMemoryFootprint(b *testing.B) {
	sizes := []int{
		100, 200, 300, 400, 500, 600, 700, 800, 900,
		1000, 1100, 1200, 1300, 1400, 1500, 1600,
		1700, 1800, 1900, 2000, 2100, 2200,
	}
	for _, n := range sizes {
		b.Run("n="+strconv.Itoa(n), func(b *testing.B) {
			b.Run("runtime map", func(b *testing.B) {
				// max load factor 6.5/8
				var m map[int]int
				for i := 0; i < b.N; i++ {
					m = make(map[int]int, n)
				}
				require.NotNil(b, m)
				b.ReportAllocs()
			})
			b.Run("swiss.Map", func(b *testing.B) {
				// max load factor 7/8
				var m *Map[int, int]
				for i := 0; i < b.N; i++ {
					m = NewMap[int, int](uint32(n))
				}
				require.NotNil(b, m)
				b.ReportAllocs()
			})
		})
	}
}

func benchmarkRuntimeMap[K comparable](b *testing.B, keys []K) {
	n := uint32(len(keys))
	mod := n - 1 // power of 2 fast modulus
	require.Equal(b, 1, bits.OnesCount32(n))
	m := make(map[K]K, n)
	for _, k := range keys {
		m[k] = k
	}
	b.ResetTimer()
	var ok bool
	for i := 0; i < b.N; i++ {
		_, ok = m[keys[uint32(i)&mod]]
	}
	assert.True(b, ok)
	b.ReportAllocs()
}

func benchmarkSwissMap[K comparable](b *testing.B, keys []K) {
	n := uint32(len(keys))
	mod := n - 1 // power of 2 fast modulus
	require.Equal(b, 1, bits.OnesCount32(n))
	m := NewMap[K, K](n)
	for _, k := range keys {
		m.Put(k, k)
	}
	b.ResetTimer()
	var ok bool
	for i := 0; i < b.N; i++ {
		_, ok = m.Get(keys[uint32(i)&mod])
	}
	assert.True(b, ok)
	b.ReportAllocs()
}

func benchmarkThepuddsMap(b *testing.B, keys []int64) {
	n := uint32(len(keys))
	mod := n - 1 // power of 2 fast modulus
	require.Equal(b, 1, bits.OnesCount32(n))
	m := swisstable.New(len(keys))
	for _, k := range keys {
		m.Set(swisstable.Key(k), swisstable.Value(k))
	}
	b.ResetTimer()
	var v swisstable.Value
	var ok bool
	for i := 0; i < b.N; i++ {
		v, ok = m.Get(swisstable.Key(keys[uint32(i)&mod]))
	}
	assert.NotNil(b, v)
	assert.True(b, ok)
}

func generateInt64Data(n int) (data []int64) {
	data = make([]int64, n)
	var x int64
	for i := range data {
		x += rand.Int63n(128) + 1
		data[i] = x
	}
	return
}
