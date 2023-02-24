package swiss

import (
	"fmt"
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
	t.Run("iter", func(t *testing.T) {
		testMapIter(t, keys)
	})
	t.Run("grow", func(t *testing.T) {
		testMapGrow(t, keys)
	})
	t.Run("probe stats", func(t *testing.T) {
		testProbeStats(t, keys)
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

func testProbeStats[K comparable](t *testing.T, keys []K) {
	runTest := func(load float32) {
		n := uint32(len(keys))
		sz, k := loadFactorSample(n, load)
		m := NewMap[K, int](sz)
		for i, key := range keys[:k] {
			m.Put(key, i)
		}
		// todo: assert stat invariants?
		stats := getProbeStats(t, m, keys)
		t.Log(fmtProbeStats(stats))
	}
	t.Run("load_factor=0.5", func(t *testing.T) {
		runTest(0.5)
	})
	t.Run("load_factor=0.75", func(t *testing.T) {
		runTest(0.75)
	})
	t.Run("load_factor=max", func(t *testing.T) {
		runTest(maxLoadFactor)
	})
}

// calculates the sample size and map size necessary to
// create a load factor of |load| given |n| data points
func loadFactorSample(n uint32, targetLoad float32) (mapSz, sampleSz uint32) {
	if targetLoad > maxLoadFactor {
		targetLoad = maxLoadFactor
	}
	// tables are assumed to be power of two
	sampleSz = uint32(float32(n) * targetLoad)
	mapSz = uint32(float32(n) * maxLoadFactor)
	return
}

type probeStats struct {
	groups     uint32
	loadFactor float32
	presentCnt uint32
	presentMin uint32
	presentMax uint32
	presentAvg float32
	absentCnt  uint32
	absentMin  uint32
	absentMax  uint32
	absentAvg  float32
}

func fmtProbeStats(s probeStats) string {
	g := fmt.Sprintf("groups=%d load=%f\n", s.groups, s.loadFactor)
	p := fmt.Sprintf("present(n=%d): min=%d max=%d avg=%f\n",
		s.presentCnt, s.presentMin, s.presentMax, s.presentAvg)
	a := fmt.Sprintf("absent(n=%d):  min=%d max=%d avg=%f\n",
		s.absentCnt, s.absentMin, s.absentMax, s.absentAvg)
	return g + p + a
}

func getProbeLength[K comparable, V any](t *testing.T, m *Map[K, V], key K) (length uint32, ok bool) {
	var end uint32
	hi, lo := hashKey(m.hash, key)
	start := probeStart(hi, len(m.groups))
	end, _, ok = m.find(key, hi, lo)
	if end < start { // wrapped
		end += uint32(len(m.groups))
	}
	length = (end - start) + 1
	require.True(t, length > 0)
	return
}

func getProbeStats[K comparable, V any](t *testing.T, m *Map[K, V], keys []K) (stats probeStats) {
	stats.groups = uint32(len(m.groups))
	stats.loadFactor = m.loadFactor()
	var presentSum, absentSum float32
	stats.presentMin = math.MaxInt32
	stats.absentMin = math.MaxInt32
	for _, key := range keys {
		l, ok := getProbeLength(t, m, key)
		if ok {
			stats.presentCnt++
			presentSum += float32(l)
			if stats.presentMin > l {
				stats.presentMin = l
			}
			if stats.presentMax < l {
				stats.presentMax = l
			}
		} else {
			stats.absentCnt++
			absentSum += float32(l)
			if stats.absentMin > l {
				stats.absentMin = l
			}
			if stats.absentMax < l {
				stats.absentMax = l
			}
		}
	}
	if stats.presentCnt == 0 {
		stats.presentMin = 0
	} else {
		stats.presentAvg = presentSum / float32(stats.presentCnt)
	}
	if stats.absentCnt == 0 {
		stats.absentMin = 0
	} else {
		stats.absentAvg = absentSum / float32(stats.absentCnt)
	}
	return
}

func TestNumGroups(t *testing.T) {
	assert.Equal(t, uint32(1), numGroups(0))
	assert.Equal(t, uint32(1), numGroups(1))
	// max load factor 14/16
	assert.Equal(t, uint32(1), numGroups(14))
	assert.Equal(t, uint32(2), numGroups(15))
	assert.Equal(t, uint32(2), numGroups(28))
	assert.Equal(t, uint32(3), numGroups(29))
	assert.Equal(t, uint32(4), numGroups(56))
	assert.Equal(t, uint32(5), numGroups(57))
}
