// Copyright 2023 Dolthub, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package swiss

import (
	"math/bits"
	"math/rand"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

func BenchmarkStringMaps(b *testing.B) {
	const keySz = 8
	benchmarkMaps(b, func(n int) []string {
		return genStringData(keySz, n)
	})
}

func BenchmarkInt64Maps(b *testing.B) {
	benchmarkMaps(b, generateInt64Data)
}

func benchmarkMaps[K comparable](b *testing.B, genData func(n int) []K) {
	sizes := []int{16, 128, 1024, 8192, 131072}
	for _, n := range sizes {
		b.Run("n="+strconv.Itoa(n), func(b *testing.B) {
			b.Run("runtime map get present", func(b *testing.B) {
				benchmarkRuntimeMapGetPresent(b, genData(n))
			})
			b.Run("swiss.Map get present", func(b *testing.B) {
				benchmarkSwissMapGetPresent(b, genData(n))
			})
			b.Run("runtime map has present", func(b *testing.B) {
				benchmarkRuntimeMapHasPresent(b, genData(n))
			})
			b.Run("swiss.Map has present", func(b *testing.B) {
				benchmarkSwissMapHasPresent(b, genData(n))
			})
			b.Run("runtime map has absent", func(b *testing.B) {
				benchmarkRuntimeMapHasAbsent(b, genData(n*2))
			})
			b.Run("swiss.Map has absent", func(b *testing.B) {
				benchmarkSwissMapHasAbsent(b, genData(n*2))
			})
			b.Run("runtime map put", func(b *testing.B) {
				benchmarkRuntimeMapPut(b, genData(n))
			})
			b.Run("swiss.Map put", func(b *testing.B) {
				benchmarkSwissMapPut(b, genData(n))
			})
		})
	}
}

func TestMemoryFootprint(t *testing.T) {
	t.Skip("unskip for memory footprint stats")
	var samples []float64
	for n := 10; n <= 50_000; n += 10 {
		b1 := testing.Benchmark(func(b *testing.B) {
			// max load factor 7/8
			m := NewMap[int, int](uint32(n))
			require.NotNil(b, m)
		})
		b2 := testing.Benchmark(func(b *testing.B) {
			// max load factor 6.5/8
			m := make(map[int]int, n)
			require.NotNil(b, m)
		})
		b3 := testing.Benchmark(func(b *testing.B) {
			m := make([][2]int, n)
			require.NotNil(b, m)
		})
		s1 := b1.MemBytes
		s2 := b2.MemBytes
		s3 := b3.MemBytes
		x := float64(s1) / float64(s2)
		t.Logf("%d,%d,%d,%d,%f", n, s1, s2, s3, x)
		samples = append(samples, x)
	}
	t.Logf("mean size ratio: %.3f", mean(samples))
}

func benchmarkRuntimeMapGetPresent[K comparable](b *testing.B, keys []K) {
	n := uint32(len(keys))
	mod := n - 1 // power of 2 fast modulus
	require.Equal(b, 1, bits.OnesCount32(n))
	m := make(map[K]K, n)
	for _, k := range keys {
		m[k] = k
	}
	b.ResetTimer()
	var val K
	var ok bool
	for i := 0; i < b.N; i++ {
		val, ok = m[keys[uint32(i)&mod]]
	}
	assert.NotNil(b, val)
	assert.True(b, ok)
	b.ReportAllocs()
}

func benchmarkSwissMapGetPresent[K comparable](b *testing.B, keys []K) {
	n := uint32(len(keys))
	mod := n - 1 // power of 2 fast modulus
	require.Equal(b, 1, bits.OnesCount32(n))
	m := NewMap[K, K](n)
	for _, k := range keys {
		m.Put(k, k)
	}
	b.ResetTimer()
	var val K
	var ok bool
	for i := 0; i < b.N; i++ {
		val, ok = m.Get(keys[uint32(i)&mod])
	}
	assert.NotNil(b, val)
	assert.True(b, ok)
	b.ReportAllocs()
}

func benchmarkRuntimeMapHasPresent[K comparable](b *testing.B, keys []K) {
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

func benchmarkSwissMapHasPresent[K comparable](b *testing.B, keys []K) {
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
		ok = m.Has(keys[uint32(i)&mod])
	}
	assert.True(b, ok)
	b.ReportAllocs()
}

func benchmarkRuntimeMapHasAbsent[K comparable](b *testing.B, data []K) {
	present, absent := data[:len(data)/2], data[len(data)/2:]
	n := uint32(len(present))
	mod := n - 1 // power of 2 fast modulus
	require.Equal(b, 1, bits.OnesCount32(n))
	m := make(map[K]K, n)
	for _, k := range present {
		m[k] = k
	}
	b.ResetTimer()
	var ok bool
	for i := 0; i < b.N; i++ {
		_, ok = m[absent[uint32(i)&mod]]
	}
	assert.False(b, ok)
	b.ReportAllocs()
}

func benchmarkSwissMapHasAbsent[K comparable](b *testing.B, data []K) {
	present, absent := data[:len(data)/2], data[len(data)/2:]
	n := uint32(len(present))
	mod := n - 1 // power of 2 fast modulus
	require.Equal(b, 1, bits.OnesCount32(n))
	m := NewMap[K, K](n)
	for _, k := range present {
		m.Put(k, k)
	}
	b.ResetTimer()
	var ok bool
	for i := 0; i < b.N; i++ {
		ok = m.Has(absent[uint32(i)&mod])
	}
	assert.False(b, ok)
	b.ReportAllocs()
}

func benchmarkRuntimeMapPut[K comparable](b *testing.B, keys []K) {
	n := uint32(len(keys))
	mod := n - 1 // power of 2 fast modulus
	require.Equal(b, 1, bits.OnesCount32(n))
	m := make(map[K]int, n)
	for i, k := range keys {
		m[k] = -i
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m[keys[uint32(i)&mod]] = i
	}
	b.ReportAllocs()
}

func benchmarkSwissMapPut[K comparable](b *testing.B, keys []K) {
	n := uint32(len(keys))
	mod := n - 1 // power of 2 fast modulus
	require.Equal(b, 1, bits.OnesCount32(n))
	m := NewMap[K, int](n)
	for i, k := range keys {
		m.Put(k, -i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Put(keys[uint32(i)&mod], i)
	}
	b.ReportAllocs()
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

func mean(samples []float64) (m float64) {
	for _, s := range samples {
		m += s
	}
	return m / float64(len(samples))
}
