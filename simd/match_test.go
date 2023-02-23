// Copyright 2022 Dolthub, Inc.
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

package simd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

//go:generate go run asm.go -out match.s -stubs match_amd64.go

func TestMatchMetadata(t *testing.T) {
	meta := [16]int8{
		0, 1, 2, 3, 4, 5, 6, 7,
		8, 9, 10, 11, 12, 13, 14, 15,
	}
	t.Run("simd match", func(t *testing.T) {
		for _, candidate := range meta {
			expected := uint16(1) << candidate
			mask := MatchMetadata(&meta, candidate)
			assert.NotZero(t, mask)
			assert.Equal(t, expected, mask)
		}
	})
	t.Run("fallback match", func(t *testing.T) {
		for _, candidate := range meta {
			expected := uint16(1) << candidate
			mask := matchMetadata(&meta, candidate)
			assert.NotZero(t, mask)
			assert.Equal(t, expected, mask)
		}
	})
}

func BenchmarkMatchMetadata(b *testing.B) {
	meta := [16]int8{
		0, 1, 2, 3, 4, 5, 6, 7,
		8, 9, 10, 11, 12, 13, 14, 15,
	}
	b.Run("simd match", func(b *testing.B) {
		var mask uint16
		for i := 0; i < b.N; i++ {
			mask = MatchMetadata(&meta, int8(i))
		}
		b.Log(mask)
	})
	b.Run("generic match", func(b *testing.B) {
		var mask uint16
		for i := 0; i < b.N; i++ {
			mask = matchMetadata(&meta, int8(i))
		}
		b.Log(mask)
	})
}

func TestCastBool(t *testing.T) {
	assert.Equal(t, int8(0), castBool(false))
	assert.Equal(t, int8(1), castBool(true))
}
