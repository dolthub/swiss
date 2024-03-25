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
	"github.com/dolthub/maphash"
)

const (
	maxLoadFactor = float32(maxAvgGroupLoad) / float32(groupSize)
)

type Size interface {
	uint32 | uint64
}

// Map is an open-addressing hash map
// based on Abseil's flat_hash_map.
type Map[K comparable, V any, S Size] struct {
	ctrl     []metadata
	groups   []group[K, V]
	hash     maphash.Hasher[K]
	resident S
	dead     S
	limit    S
}

// metadata is the h2 metadata array for a group.
// find operations first probe the controls bytes
// to filter candidates before matching keys
type metadata [groupSize]int8

// group is a group of 16 key-value pairs
type group[K comparable, V any] struct {
	keys   [groupSize]K
	values [groupSize]V
}

const (
	h1Mask    uint64 = 0xffff_ffff_ffff_ff80
	h2Mask    uint64 = 0x0000_0000_0000_007f
	empty     int8   = -128 // 0b1000_0000
	tombstone int8   = -2   // 0b1111_1110
)

// h1 is a 57 bit hash prefix
type h1 uint64

// h2 is a 7 bit hash suffix
type h2 int8

// NewMap constructs a Map.
func NewMap[K comparable, V any](sz uint32) (m *Map[K, V, uint32]) {
	return newMap[K, V, uint32](sz)
}

// NewMap constructs a Map.
func NewMap64[K comparable, V any](sz uint64) (m *Map[K, V, uint64]) {
	return newMap[K, V, uint64](sz)
}

func newMap[K comparable, V any, S Size](sz S) (m *Map[K, V, S]) {
	groups := numGroups(sz)
	m = &Map[K, V, S]{
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
func (m *Map[K, V, S]) Has(key K) (ok bool) {
	hi, lo := splitHash(m.hash.Hash(key))
	g := probeStart[S](hi, len(m.groups))
	for { // inlined find loop
		matches := metaMatchH2(&m.ctrl[g], lo)
		for matches != 0 {
			s := nextMatch[S](&matches)
			if key == m.groups[g].keys[s] {
				ok = true
				return
			}
		}
		// |key| is not in group |g|,
		// stop probing if we see an empty slot
		matches = metaMatchEmpty(&m.ctrl[g])
		if matches != 0 {
			ok = false
			return
		}
		g += 1 // linear probing
		if g >= S(len(m.groups)) {
			g = 0
		}
	}
}

// Get returns the |value| mapped by |key| if one exists.
func (m *Map[K, V, S]) Get(key K) (value V, ok bool) {
	hi, lo := splitHash(m.hash.Hash(key))
	g := probeStart[S](hi, len(m.groups))
	for { // inlined find loop
		matches := metaMatchH2(&m.ctrl[g], lo)
		for matches != 0 {
			s := nextMatch[S](&matches)
			if key == m.groups[g].keys[s] {
				value, ok = m.groups[g].values[s], true
				return
			}
		}
		// |key| is not in group |g|,
		// stop probing if we see an empty slot
		matches = metaMatchEmpty(&m.ctrl[g])
		if matches != 0 {
			ok = false
			return
		}
		g += 1 // linear probing
		if g >= S(len(m.groups)) {
			g = 0
		}
	}
}

// Put attempts to insert |key| and |value|
func (m *Map[K, V, S]) Put(key K, value V) {
	if m.resident >= m.limit {
		m.rehash(m.nextSize())
	}
	hi, lo := splitHash(m.hash.Hash(key))
	g := probeStart[S](hi, len(m.groups))
	for { // inlined find loop
		matches := metaMatchH2(&m.ctrl[g], lo)
		for matches != 0 {
			s := nextMatch[S](&matches)
			if key == m.groups[g].keys[s] { // update
				m.groups[g].keys[s] = key
				m.groups[g].values[s] = value
				return
			}
		}
		// |key| is not in group |g|,
		// stop probing if we see an empty slot
		matches = metaMatchEmpty(&m.ctrl[g])
		if matches != 0 { // insert
			s := nextMatch[S](&matches)
			m.groups[g].keys[s] = key
			m.groups[g].values[s] = value
			m.ctrl[g][s] = int8(lo)
			m.resident++
			return
		}
		g += 1 // linear probing
		if g >= S(len(m.groups)) {
			g = 0
		}
	}
}

// Delete attempts to remove |key|, returns true successful.
func (m *Map[K, V, S]) Delete(key K) (ok bool) {
	hi, lo := splitHash(m.hash.Hash(key))
	g := probeStart[S](hi, len(m.groups))
	for {
		matches := metaMatchH2(&m.ctrl[g], lo)
		for matches != 0 {
			s := nextMatch[S](&matches)
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
				var k K
				var v V
				m.groups[g].keys[s] = k
				m.groups[g].values[s] = v
				return
			}
		}
		// |key| is not in group |g|,
		// stop probing if we see an empty slot
		matches = metaMatchEmpty(&m.ctrl[g])
		if matches != 0 { // |key| absent
			ok = false
			return
		}
		g += 1 // linear probing
		if g >= S(len(m.groups)) {
			g = 0
		}
	}
}

// Iter iterates the elements of the Map, passing them to the callback.
// It guarantees that any key in the Map will be visited only once, and
// for un-mutated Maps, every key will be visited once. If the Map is
// Mutated during iteration, mutations will be reflected on return from
// Iter, but the set of keys visited by Iter is non-deterministic.
func (m *Map[K, V, S]) Iter(cb func(k K, v V) (stop bool)) {
	// take a consistent view of the table in case
	// we rehash during iteration
	ctrl, groups := m.ctrl, m.groups
	// pick a random starting group
	g := S(randIntN(len(groups)))
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
		if g >= S(len(groups)) {
			g = 0
		}
	}
}

// Clear removes all elements from the Map.
func (m *Map[K, V, S]) Clear() {
	for i, c := range m.ctrl {
		for j := range c {
			m.ctrl[i][j] = empty
		}
	}
	var k K
	var v V
	for i := range m.groups {
		g := &m.groups[i]
		for i := range g.keys {
			g.keys[i] = k
			g.values[i] = v
		}
	}
	m.resident, m.dead = 0, 0
}

// Count returns the number of elements in the Map.
func (m *Map[K, V, S]) Count() int {
	return int(m.resident - m.dead)
}

// Capacity returns the number of additional elements
// the can be added to the Map before resizing.
func (m *Map[K, V, S]) Capacity() int {
	return int(m.limit - m.resident)
}

// find returns the location of |key| if present, or its insertion location if absent.
// for performance, find is manually inlined into public methods.
func (m *Map[K, V, S]) find(key K, hi h1, lo h2) (g, s S, ok bool) {
	g = probeStart[S](hi, len(m.groups))
	for {
		matches := metaMatchH2(&m.ctrl[g], lo)
		for matches != 0 {
			s = nextMatch[S](&matches)
			if key == m.groups[g].keys[s] {
				return g, s, true
			}
		}
		// |key| is not in group |g|,
		// stop probing if we see an empty slot
		matches = metaMatchEmpty(&m.ctrl[g])
		if matches != 0 {
			s = nextMatch[S](&matches)
			return g, s, false
		}
		g += 1 // linear probing
		if g >= S(len(m.groups)) {
			g = 0
		}
	}
}

func (m *Map[K, V, S]) nextSize() (n S) {
	n = S(len(m.groups)) * 2
	if m.dead >= (m.resident / 2) {
		n = S(len(m.groups))
	}
	return
}

func (m *Map[K, V, S]) rehash(n S) {
	groups, ctrl := m.groups, m.ctrl
	m.groups = make([]group[K, V], n)
	m.ctrl = make([]metadata, n)
	for i := range m.ctrl {
		m.ctrl[i] = newEmptyMetadata()
	}
	m.hash = maphash.NewSeed(m.hash)
	m.limit = n * maxAvgGroupLoad
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

func (m *Map[K, V, S]) loadFactor() float32 {
	slots := float32(len(m.groups) * groupSize)
	return float32(m.resident-m.dead) / slots
}

// numGroups returns the minimum number of groups needed to store |n| elems.
func numGroups[S Size](n S) (groups S) {
	groups = (n + maxAvgGroupLoad - 1) / maxAvgGroupLoad
	if groups == 0 {
		groups = 1
	}
	return
}

func newEmptyMetadata() (meta metadata) {
	for i := range meta {
		meta[i] = empty
	}
	return
}

func splitHash(h uint64) (h1, h2) {
	return h1((h & h1Mask) >> 7), h2(h & h2Mask)
}

func probeStart[S Size](hi h1, groups int) S {
	return fastModN(S(hi), S(groups))
}

// lemire.me/blog/2016/06/27/a-fast-alternative-to-the-modulo-reduction/
func fastModN[S Size](x, n S) S {
	return S((uint64(x) * uint64(n)) >> 32)
}
