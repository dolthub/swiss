package swiss

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/thepudds/swisstable"

	"github.com/stretchr/testify/assert"
)

func NewStringMap(sz uint32) *Map[string, string] {
	return NewMap[string, string](sz)
}

func BenchmarkMaps(b *testing.B) {
	sizes := []int{8, 64}
	counts := []int{10, 100, 1000, 10_000}
	for _, s := range sizes {
		for _, c := range counts {
			benchmarkSwissAndBuiltin(b, s, c)
		}
	}
}

func BenchmarkLargeMap(b *testing.B) {
	benchmarkSwissAndBuiltin(b, 16, 100_000)
}

func benchmarkSwissAndBuiltin(b *testing.B, keySz, count int) {
	keys := genStringData(keySz, count)
	nm := fmt.Sprintf("benchmark swiss map (count=%d,keysize=%d)", count, keySz)
	b.Run(nm, func(b *testing.B) {
		m := NewStringMap(uint32(count))
		for _, k := range keys {
			m.Put(k, k)
		}
		b.ResetTimer()
		var v string
		var ok bool
		for i := 0; i < b.N; i++ {
			v, ok = m.Get(keys[i%count])
		}
		assert.NotNil(b, v)
		assert.True(b, ok)
		b.ReportAllocs()
	})
	nm = fmt.Sprintf("benchmark go map (count=%d,keysize=%d", count, keySz)
	b.Run(nm, func(b *testing.B) {
		m := make(map[string]string, count)
		for _, k := range keys {
			m[k] = k
		}
		b.ResetTimer()
		var v string
		var ok bool
		for i := 0; i < b.N; i++ {
			v, ok = m[keys[i%count]]
		}
		assert.NotNil(b, v)
		assert.True(b, ok)
		b.ReportAllocs()
	})
}

func BenchmarkMaps2(b *testing.B) {
	getInt64Data := func(c int) (data []int64) {
		data = make([]int64, c)
		for i := range data {
			data[i] = rand.Int63()
		}
		return
	}

	counts := []int{10, 100, 1000, 10_000}
	for _, c := range counts {
		keys := getInt64Data(c)
		nm := fmt.Sprintf("benchmark thepudds map (resident=%d", c)
		b.Run(nm, func(b *testing.B) {
			m := swisstable.New(c)
			for _, k := range keys {
				m.Set(swisstable.Key(k), swisstable.Value(k))
			}
			b.ResetTimer()
			var v swisstable.Value
			var ok bool
			for i := 0; i < b.N; i++ {
				v, ok = m.Get(swisstable.Key(keys[i%c]))
			}
			assert.NotNil(b, v)
			assert.True(b, ok)
		})
		nm = fmt.Sprintf("benchmark go map (resident=%d", c)
		b.Run(nm, func(b *testing.B) {
			m := make(map[int64]int64, c)
			for _, k := range keys {
				m[k] = k
			}
			b.ResetTimer()
			var v int64
			var ok bool
			for i := 0; i < b.N; i++ {
				v, ok = m[keys[i%c]]
			}
			assert.NotNil(b, v)
			assert.True(b, ok)
		})
	}
}
