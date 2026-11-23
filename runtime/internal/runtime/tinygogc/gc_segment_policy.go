/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *     http://www.apache.org/licenses/LICENSE-2.0
 */

package tinygogc

// Budget of the heap segments the collector takes from an allocator outside
// its statically reserved heap. Target code feeds the policy the allocator and
// the collector's block geometry (gc_segment_idf.go); everything here is plain
// arithmetic so it can be unit tested on the host.
const (
	extHeapMinFree      = 256 << 10 // a smaller free size means no external heap
	extHeapFirstSegment = 1 << 20   // segments double from this size ...
	extHeapMaxSegment   = 4 << 20   // ... up to this size
	extHeapMinSegment   = 64 << 10  // a failed request is not retried below this
)

// extHeapOps is the allocator behind the external heap segments. Passing it as
// a value lets a host test drive the policy with a simulated allocator.
type extHeapOps struct {
	malloc func(size uintptr) uintptr
}

// extHeap tracks the external memory spent on heap segments.
type extHeap struct {
	bytesPerBlock      uintptr
	blocksPerStateByte uintptr

	probed   bool
	disabled bool
	capacity uintptr // bytes of external memory the segments may use in total
	reserved uintptr // bytes requested from the allocator so far
	segments int
}

// probe settles the budget from the free size and the configured limit: 0
// disables the external heap, ^uintptr(0) leaves three quarters of the free
// memory to the collector (the rest is for the system and the application),
// anything else is a byte count. The budget is fixed once, so memory that
// other components allocate later cannot raise the collector's share.
func (h *extHeap) probe(free, limit uintptr) {
	h.probed = true
	if free < extHeapMinFree {
		h.disabled = true
		return
	}
	switch limit {
	case 0:
		h.disabled = true
	case ^uintptr(0):
		h.capacity = free - free/4
	default:
		h.capacity = limit
	}
}

// request returns the size of the next segment and the smallest size that
// still serves an allocation of minimum bytes. Both are 0 when the remaining
// budget cannot hold that allocation.
func (h *extHeap) request(minimum uintptr) (size, required uintptr) {
	if !h.probed || h.disabled {
		return 0, 0
	}
	// A segment carries its block-state bytes and loses up to one block at
	// each end to alignment, so it must be larger than the data it holds.
	extra := minimum/(h.bytesPerBlock*h.blocksPerStateByte) + 2*h.bytesPerBlock
	if minimum > ^uintptr(0)-extra-(h.bytesPerBlock-1) {
		return 0, 0
	}
	required = alignUpToBlock(minimum+extra, h.bytesPerBlock)
	shift := h.segments
	if shift > 2 {
		shift = 2
	}
	size = uintptr(extHeapFirstSegment) << shift
	if size < required {
		size = required
	}
	if size > h.capacity-h.reserved {
		size = h.capacity - h.reserved
	}
	if size < required {
		return 0, 0
	}
	return size, required
}

// allocate takes the next segment from ops and returns its block-aligned data
// range. A request the allocator cannot satisfy is halved, which helps when
// the external memory is fragmented. It returns 0, 0 when no segment large
// enough for minimum could be obtained.
func (h *extHeap) allocate(minimum uintptr, ops extHeapOps) (start, end uintptr) {
	size, required := h.request(minimum)
	if size == 0 {
		return 0, 0
	}
	for {
		if addr := ops.malloc(size); addr != 0 {
			start = alignUpToBlock(addr, h.bytesPerBlock)
			end = alignDownToBlock(addr+size, h.bytesPerBlock)
			// addHeapSegment carves the state bytes out of the range and
			// needs whole blocks on both sides of them. Unreachable unless
			// addr+size wraps around; the block is leaked by design (nothing
			// else can be done with it at this point).
			if end-start < 2*h.bytesPerBlock {
				return 0, 0
			}
			h.reserved += size
			h.segments++
			return start, end
		}
		size /= 2
		if size < extHeapMinSegment || size < required {
			return 0, 0
		}
	}
}

func alignUpToBlock(addr, block uintptr) uintptr {
	return (addr + block - 1) &^ (block - 1)
}

func alignDownToBlock(addr, block uintptr) uintptr {
	return addr &^ (block - 1)
}
