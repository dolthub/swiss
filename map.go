package swiss

import (
	"math"
	"math/bits"
	"math/rand"

	"github.com/dolthub/maphash"
)

const (
	groupSize       = 16
	maxAvgGroupLoad = 14
	maxLoadFactor   = float32(maxAvgGroupLoad) / float32(groupSize)
)

// Map is an open-addressing hash map
// based on Abseil's flat_hash_map.
type Map[K comparable, V any] struct {
	ctrl     []metadata
	groups   []group[K, V]
	hash     maphash.Hasher[K]
	resident uint32
	dead     uint32
	limit    uint32
}

// group is a group of 16 key-value pairs
type group[K comparable, V any] struct {
	keys   [groupSize]K
	values [groupSize]V
}

// NewMap constructs a Map.
func NewMap[K comparable, V any](sz uint32) (m *Map[K, V]) {
	groups := numGroups(sz)
	m = &Map[K, V]{
		ctrl:   make([]metadata, groups),
		groups: make([]group[K, V], groups),
		hash:   maphash.NewHasher[K](),
		limit:  groups * maxAvgGroupLoad,
	}
	for i := range m.ctrl {
		m.ctrl[i] = newEmptyMetadata()
	}
	return
}

// Has returns true if |key| is present in |m|.
func (m *Map[K, V]) Has(key K) (ok bool) {
	hi, lo := splitHash(m.hash.Hash(key))
	g := probeStart(hi, len(m.groups))
	for { // inlined find loop
		matches := metaMatchH2(&m.ctrl[g], lo)
		for matches != 0 {
			s := uint32(bits.TrailingZeros16(uint16(matches)))
			if key == m.groups[g].keys[s] {
				ok = true
				return
			}
			matches &= ^(1 << s) // clear bit |s|
		}
		// |key| is not in group |g|,
		// stop probing if we see an empty slot
		matches = metaMatchEmpty(&m.ctrl[g])
		if matches != 0 {
			ok = false
			return
		}
		g += 1 // linear probing
		if g >= uint32(len(m.groups)) {
			g = 0
		}
	}
}

// Get returns the |value| mapped by |key| if one exists.
func (m *Map[K, V]) Get(key K) (value V, ok bool) {
	hi, lo := splitHash(m.hash.Hash(key))
	g := probeStart(hi, len(m.groups))
	for { // inlined find loop
		matches := metaMatchH2(&m.ctrl[g], lo)
		for matches != 0 {
			s := uint32(bits.TrailingZeros16(uint16(matches)))
			if key == m.groups[g].keys[s] {
				value, ok = m.groups[g].values[s], true
				return
			}
			matches &= ^(1 << s) // clear bit |s|
		}
		// |key| is not in group |g|,
		// stop probing if we see an empty slot
		matches = metaMatchEmpty(&m.ctrl[g])
		if matches != 0 {
			ok = false
			return
		}
		g += 1 // linear probing
		if g >= uint32(len(m.groups)) {
			g = 0
		}
	}
}

// Put attempts to insert |key| and |value|
func (m *Map[K, V]) Put(key K, value V) {
	if m.resident >= m.limit {
		m.rehash(m.nextSize())
	}
	hi, lo := splitHash(m.hash.Hash(key))
	g := probeStart(hi, len(m.groups))
	for { // inlined find loop
		matches := metaMatchH2(&m.ctrl[g], lo)
		for matches != 0 {
			s := uint32(bits.TrailingZeros16(uint16(matches)))
			if key == m.groups[g].keys[s] { // update
				m.groups[g].keys[s] = key
				m.groups[g].values[s] = value
				return
			}
			matches &= ^(1 << s) // clear bit |s|
		}
		// |key| is not in group |g|,
		// stop probing if we see an empty slot
		matches = metaMatchEmpty(&m.ctrl[g])
		if matches != 0 { // insert
			s := uint32(bits.TrailingZeros16(uint16(matches)))
			m.groups[g].keys[s] = key
			m.groups[g].values[s] = value
			m.ctrl[g][s] = int8(lo)
			m.resident++
			return
		}
		g += 1 // linear probing
		if g >= uint32(len(m.groups)) {
			g = 0
		}
	}
}

// Delete attempts to remove |key|, returns true successful.
func (m *Map[K, V]) Delete(key K) (ok bool) {
	hi, lo := splitHash(m.hash.Hash(key))
	g := probeStart(hi, len(m.groups))
	for {
		matches := metaMatchH2(&m.ctrl[g], lo)
		for matches != 0 {
			s := uint32(bits.TrailingZeros16(uint16(matches)))
			if key == m.groups[g].keys[s] {
				ok = true
				// optimization: if |m.ctrl[g]| contains any empty
				// metadata bytes, we can physically delete |key|
				// rather than placing a tombstone.
				// The observation is that any probes into group |g|
				// would already be terminated by the existing empty
				// slot, and therefore reclaiming slot |s| will not
				// cause premature termination of probes into |g|.
				if metaMatchEmpty(&m.ctrl[g]) != 0 {
					m.ctrl[g][s] = empty
					m.resident--
				} else {
					m.ctrl[g][s] = tombstone
					m.dead++
				}
				return
			}
			matches &= ^(1 << s) // clear bit |s|
		}
		// |key| is not in group |g|,
		// stop probing if we see an empty slot
		matches = metaMatchEmpty(&m.ctrl[g])
		if matches != 0 { // |key| absent
			ok = false
			return
		}
		g += 1 // linear probing
		if g >= uint32(len(m.groups)) {
			g = 0
		}
	}
}

// Iter iterates the elements of the Map, passing them to the callback.
// It guarantees that any key in the Map will be visited only once, and
// for un-mutated Maps, every key will be visited once. If the Map is
// Mutated during iteration, mutations will be reflected on return from
// Iter, but the set of keys visited by Iter is non-deterministic.
func (m *Map[K, V]) Iter(cb func(k K, v V) (stop bool)) {
	// take a consistent view of the table in case
	// we rehash during iteration
	ctrl, groups := m.ctrl, m.groups
	// pick a random starting group
	g := rand.Intn(len(groups))
	for n := 0; n < len(groups); n++ {
		for s, c := range ctrl[g] {
			if c == empty || c == tombstone {
				continue
			}
			k, v := groups[g].keys[s], groups[g].values[s]
			if stop := cb(k, v); stop {
				return
			}
		}
		g++
		if g >= len(groups) {
			g = 0
		}
	}
}

// Count returns the number of elements in the Map.
func (m *Map[K, V]) Count() int {
	return int(m.resident - m.dead)
}

// find returns the location of |key| if present, or its insertion location if absent.
// for performance, find is manually inlined into public methods.
func (m *Map[K, V]) find(key K, hi h1, lo h2) (g, s uint32, ok bool) {
	g = probeStart(hi, len(m.groups))
	for {
		matches := metaMatchH2(&m.ctrl[g], lo)
		for matches != 0 {
			s = uint32(bits.TrailingZeros16(uint16(matches)))
			if key == m.groups[g].keys[s] {
				return g, s, true
			}
			matches &= ^(1 << s) // clear bit |s|
		}
		// |key| is not in group |g|,
		// stop probing if we see an empty slot
		matches = metaMatchEmpty(&m.ctrl[g])
		if matches != 0 {
			s = uint32(bits.TrailingZeros16(uint16(matches)))
			return g, s, false
		}
		g += 1 // linear probing
		if g >= uint32(len(m.groups)) {
			g = 0
		}
	}
}

func (m *Map[K, V]) nextSize() (n uint32) {
	n = uint32(len(m.groups)) * 2
	if m.dead >= (m.resident / 2) {
		n = uint32(len(m.groups))
	}
	return
}

func (m *Map[K, V]) rehash(n uint32) {
	groups, ctrl := m.groups, m.ctrl
	m.groups = make([]group[K, V], n)
	m.ctrl = make([]metadata, n)
	for i := range m.ctrl {
		m.ctrl[i] = newEmptyMetadata()
	}
	m.hash = maphash.NewHasher[K]()
	m.limit = n * groupSize
	m.resident, m.dead = 0, 0
	for g := range ctrl {
		for s := range ctrl[g] {
			c := ctrl[g][s]
			if c == empty || c == tombstone {
				continue
			}
			m.Put(groups[g].keys[s], groups[g].values[s])
		}
	}
}

func (m *Map[K, V]) loadFactor() float32 {
	slots := float32(len(m.groups) * groupSize)
	return float32(m.resident-m.dead) / slots
}

func probeStart(hi h1, groups int) uint32 {
	return fastModN(uint32(hi), uint32(groups))

}

// numGroups returns the minimum number of groups needed to store |n|
// elements, accounting for load factor and power of 2 alignment
func numGroups(n uint32) (groups uint32) {
	groups = uint32(math.Ceil(float64(n) / float64(maxAvgGroupLoad)))
	if groups == 0 {
		groups = 1
	}
	return
}
