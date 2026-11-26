//go:build llgo && baremetal && tinygo.riscv32 && llgo.idf && !nogc

/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package tinygogc

import _ "unsafe" // for go:linkname

// ESP-IDF manages the memory around the collector's static heap, so the heap
// cannot be extended in place. It grows with segments taken from the SPIRAM
// heap instead; a board without SPIRAM keeps the static heap only.
const segmentedHeap = true

// esp_heap_caps.h: external memory only, and byte-addressable.
const (
	mallocCap8Bit   = 1 << 2
	mallocCapSPIRAM = 1 << 10
)

var idfExtHeap = extHeap{bytesPerBlock: bytesPerBlock, blocksPerStateByte: blocksPerStateByte}

func newHeapSegment(minimum uintptr) (uintptr, uintptr) {
	if !idfExtHeap.probed {
		idfExtHeap.probe(heapCapsGetFreeSize(mallocCapSPIRAM), llgoGCExtHeapCap)
	}
	start, end := idfExtHeap.allocate(minimum, extHeapOps{malloc: mallocSPIRAM})
	if start != 0 {
		// Written before addHeapSegment registers the segment, so the C-side
		// ledger may briefly lead the collector's own view by one segment.
		llgoGCExtBytes = idfExtHeap.reserved
	}
	return start, end
}

func mallocSPIRAM(size uintptr) uintptr {
	return heapCapsMalloc(size, mallocCapSPIRAM|mallocCap8Bit)
}

// HeapStats reports the size of the Go heap: usable data bytes, reserved
// bytes including block-state bytes, the number of segments, and the reserved
// bytes taken from SPIRAM. It only reads counters and never collects. Safe
// under the current single-task scheduler invariant: allocation never yields
// between growHeap and addHeapSegment, so the counters cannot be observed
// mid-update. Revisit if a multi-task allocation path is ever introduced.
func HeapStats() (usable, reserved uintptr, segments int, external uintptr) {
	return heapUsableSize(), heapReservedSize(), heapSegmentCount, idfExtHeap.reserved
}

// Both take and release a spinlock only; they never block, which the
// allocation path inside the collector requires.
//
//go:linkname heapCapsMalloc C.heap_caps_malloc
func heapCapsMalloc(size, caps uintptr) uintptr

//go:linkname heapCapsGetFreeSize C.heap_caps_get_free_size
func heapCapsGetFreeSize(caps uintptr) uintptr

// Budget of the SPIRAM heap. The weak definition in _wrap/psram_idf.c makes
// it 75% of the free size at the first growth; an application overrides it
// with a byte count, or with 0 to disable the SPIRAM heap.
//
//go:linkname llgoGCExtHeapCap C.llgo_gc_ext_heap_cap
var llgoGCExtHeapCap uintptr

// SPIRAM bytes reserved by heap segments so far, written after each successful
// segment so that C code can report the same figure as HeapStats.
//
//go:linkname llgoGCExtBytes C.llgo_gc_ext_bytes
var llgoGCExtBytes uintptr
