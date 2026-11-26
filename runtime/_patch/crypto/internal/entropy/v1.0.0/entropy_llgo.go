// Copyright 2025 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSES/Go-BSD-3-Clause.txt
// file at this module root for license terms.

package entropy

//llgo:skip ScratchBuffer Sample

import (
	"crypto/internal/fips140deps/time"
)

// ScratchBuffer is the timing-noise scratch buffer of the SP 800-90B entropy
// source. The stdlib declares it as 32 MiB of static .bss so that the access
// pattern spreads over a huge address range; embedded targets cannot afford
// that, so LLGo shrinks it to 64 KiB. Its contents do not matter.
type ScratchBuffer [1 << 16]byte

// scratchBits mirrors log2(len(ScratchBuffer{})) for the index mask in Sample.
const scratchBits = 16

// Sample is the stdlib Sample with the index mask adapted to the shrunken
// ScratchBuffer: the stdlib version masks the LCG output with 1<<25-1 and
// would write out of bounds against the 64 KiB buffer.
func (s *source) Sample() uint8 {
	memory, lcgState := s.memory, s.lcgState
	if memory == nil { // remove the nil check from the inlined touchMemory calls
		panic("entropy: nil memory buffer")
	}
	for range 64 {
		lcgState = 1664525*lcgState + 1013904223
		// Discard the lower bits, which tend to fall into short cycles.
		idx := (lcgState >> 6) & (1<<scratchBits - 1)
		touchMemory(memory, idx)
	}
	s.lcgState = lcgState

	t := time.HighPrecisionNow()
	sample := t - s.previous
	s.previous = t

	// Reduce the symbol space to 256 values, assuming most of the entropy is
	// in the least-significant bits, which represent the highest-resolution
	// timing differences.
	return uint8(sample)
}

// touchMemory stays as declared by the stdlib file; the Sample above keeps
// every index inside the shrunken buffer, and the index is 4-aligned before
// use, so the uint32 write there remains in bounds.
