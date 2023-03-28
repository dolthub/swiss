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

//go:build ignore
// +build ignore

package main

import (
	. "github.com/mmcloughlin/avo/build"
	. "github.com/mmcloughlin/avo/operand"
)

func main() {
	ConstraintExpr("amd64")

	TEXT("MatchMetadata", NOSPLIT, "func(metadata *[16]int8, hash int8) uint16")
	Doc("MatchMetadata performs a 16-way probe of |metadata| using SSE instructions",
		"nb: |metadata| must be an aligned pointer")
	m := Mem{Base: Load(Param("metadata"), GP64())}
	h := Load(Param("hash"), GP32())
	mask := GP32()

	x0, x1, x2 := XMM(), XMM(), XMM()
	MOVD(h, x0)
	PXOR(x1, x1)
	PSHUFB(x1, x0)
	MOVOU(m, x2)
	PCMPEQB(x2, x0)
	PMOVMSKB(x0, mask)

	Store(mask.As16(), ReturnIndex(0))
	RET()
	Generate()
}
