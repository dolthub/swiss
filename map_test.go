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
	"math"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

func TestSwissMap(t *testing.T) {
	t.Run("strings=100", func(t *testing.T) {
		testSwissMap(t, genStringData(16, 100))
	})
	t.Run("strings=1000", func(t *testing.T) {
		testSwissMap(t, genStringData(16, 1000))
	})
	t.Run("strings=10_000", func(t *testing.T) {
		testSwissMap(t, genStringData(16, 10_000))
	})
	t.Run("strings=100_000", func(t *testing.T) {
		testSwissMap(t, genStringData(16, 100_000))
	})
	t.Run("uint32=100", func(t *testing.T) {
		testSwissMap(t, genUint32Data(100))
	})
	t.Run("uint32=1000", func(t *testing.T) {
		testSwissMap(t, genUint32Data(1000))
	})
	t.Run("uint32=10_000", func(t *testing.T) {
		testSwissMap(t, genUint32Data(10_000))
	})
	t.Run("uint32=100_000", func(t *testing.T) {
		testSwissMap(t, genUint32Data(100_000))
	})
	t.Run("string capacity", func(t *testing.T) {
		testSwissMapCapacity(t, func(n int) []string {
			return genStringData(16, n)
		})
	})
	t.Run("uint32 capacity", func(t *testing.T) {
		testSwissMapCapacity(t, genUint32Data)
	})
}

func testSwissMap[K comparable](t *testing.T, keys []K) {
	// sanity check
	require.Equal(t, len(keys), len(uniq(keys)), keys)
	t.Run("put", func(t *testing.T) {
		testMapPut(t, keys)
	})
	t.Run("has", func(t *testing.T) {
		testMapHas(t, keys)
	})
	t.Run("get", func(t *testing.T) {
		testMapGet(t, keys)
	})
	t.Run("delete", func(t *testing.T) {
		testMapDelete(t, keys)
	})
	t.Run("clear", func(t *testing.T) {
		testMapClear(t, keys)
	})
	t.Run("iter", func(t *testing.T) {
		testMapIter(t, keys)
	})
	t.Run("grow", func(t *testing.T) {
		testMapGrow(t, keys)
	})
}

func uniq[K comparable](keys []K) []K {
	s := make(map[K]struct{}, len(keys))
	for _, k := range keys {
		s[k] = struct{}{}
	}
	u := make([]K, 0, len(keys))
	for k := range s {
		u = append(u, k)
	}
	return u
}

func genStringData(size, count int) (keys []string) {
	src := rand.New(rand.NewSource(int64(size * count)))
	letters := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	r := make([]rune, size*count)
	for i := range r {
		r[i] = letters[src.Intn(len(letters))]
	}
	keys = make([]string, count)
	for i := range keys {
		keys[i] = string(r[:size])
		r = r[size:]
	}
	return
}

func genUint32Data(count int) (keys []uint32) {
	keys = make([]uint32, count)
	var x uint32
	for i := range keys {
		x += (rand.Uint32() % 128) + 1
		keys[i] = x
	}
	return
}

func testMapPut[K comparable](t *testing.T, keys []K) {
	m := NewMap[K, int](uint32(len(keys)))
	assert.Equal(t, 0, m.Count())
	for i, key := range keys {
		m.Put(key, i)
	}
	assert.Equal(t, len(keys), m.Count())
	// overwrite
	for i, key := range keys {
		m.Put(key, -i)
	}
	assert.Equal(t, len(keys), m.Count())
	for i, key := range keys {
		act, ok := m.Get(key)
		assert.True(t, ok)
		assert.Equal(t, -i, act)
	}
	assert.Equal(t, len(keys), int(m.resident))
}

func testMapHas[K comparable](t *testing.T, keys []K) {
	m := NewMap[K, int](uint32(len(keys)))
	for i, key := range keys {
		m.Put(key, i)
	}
	for _, key := range keys {
		ok := m.Has(key)
		assert.True(t, ok)
	}
}

func testMapGet[K comparable](t *testing.T, keys []K) {
	m := NewMap[K, int](uint32(len(keys)))
	for i, key := range keys {
		m.Put(key, i)
	}
	for i, key := range keys {
		act, ok := m.Get(key)
		assert.True(t, ok)
		assert.Equal(t, i, act)
	}
}

func testMapDelete[K comparable](t *testing.T, keys []K) {
	m := NewMap[K, int](uint32(len(keys)))
	assert.Equal(t, 0, m.Count())
	for i, key := range keys {
		m.Put(key, i)
	}
	assert.Equal(t, len(keys), m.Count())
	for _, key := range keys {
		m.Delete(key)
		ok := m.Has(key)
		assert.False(t, ok)
	}
	assert.Equal(t, 0, m.Count())
}

func testMapClear[K comparable](t *testing.T, keys []K) {
	m := NewMap[K, int](0)
	assert.Equal(t, 0, m.Count())
	for i, key := range keys {
		m.Put(key, i)
	}
	assert.Equal(t, len(keys), m.Count())
	m.Clear()
	assert.Equal(t, 0, m.Count())
	for _, key := range keys {
		ok := m.Has(key)
		assert.False(t, ok)
		_, ok = m.Get(key)
		assert.False(t, ok)
	}
	var calls int
	m.Iter(func(k K, v int) (stop bool) {
		calls++
		return
	})
	assert.Equal(t, 0, calls)
}

func testMapIter[K comparable](t *testing.T, keys []K) {
	m := NewMap[K, int](uint32(len(keys)))
	for i, key := range keys {
		m.Put(key, i)
	}
	visited := make(map[K]uint, len(keys))
	for _, k := range keys {
		visited[k] = 0
	}
	m.Iter(func(k K, v int) (stop bool) {
		visited[k]++
		return
	})
	for _, c := range visited {
		assert.Equal(t, c, uint(1))
	}
	// mutate on iter
	m.Iter(func(k K, v int) (stop bool) {
		m.Put(k, -v)
		return
	})
	for i, key := range keys {
		act, ok := m.Get(key)
		assert.True(t, ok)
		assert.Equal(t, -i, act)
	}
}

func testMapGrow[K comparable](t *testing.T, keys []K) {
	n := uint32(len(keys))
	m := NewMap[K, int](n / 10)
	for i, key := range keys {
		m.Put(key, i)
	}
	for i, key := range keys {
		act, ok := m.Get(key)
		assert.True(t, ok)
		assert.Equal(t, i, act)
	}
}

func testSwissMapCapacity[K comparable](t *testing.T, gen func(n int) []K) {
	// Capacity() behavior depends on |groupSize|
	// which varies by processor architecture.
	caps := []uint32{
		1 * maxAvgGroupLoad,
		2 * maxAvgGroupLoad,
		3 * maxAvgGroupLoad,
		4 * maxAvgGroupLoad,
		5 * maxAvgGroupLoad,
		10 * maxAvgGroupLoad,
		25 * maxAvgGroupLoad,
		50 * maxAvgGroupLoad,
		100 * maxAvgGroupLoad,
	}
	for _, c := range caps {
		m := NewMap[K, K](c)
		assert.Equal(t, int(c), m.Capacity())
		keys := gen(rand.Intn(int(c)))
		for _, k := range keys {
			m.Put(k, k)
		}
		assert.Equal(t, int(c)-len(keys), m.Capacity())
		assert.Equal(t, int(c), m.Count()+m.Capacity())
	}
}

func TestNumGroups(t *testing.T) {
	assert.Equal(t, expected(0), numGroups(0))
	assert.Equal(t, expected(1), numGroups(1))
	// max load factor 0.875
	assert.Equal(t, expected(14), numGroups(14))
	assert.Equal(t, expected(15), numGroups(15))
	assert.Equal(t, expected(28), numGroups(28))
	assert.Equal(t, expected(29), numGroups(29))
	assert.Equal(t, expected(56), numGroups(56))
	assert.Equal(t, expected(57), numGroups(57))
}

func expected(x int) (groups uint32) {
	groups = uint32(math.Ceil(float64(x) / float64(maxAvgGroupLoad)))
	if groups == 0 {
		groups = 1
	}
	return
}
