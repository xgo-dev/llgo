//go:build llgo && wasip1 && wasm && llgo.wasi_threads && llgo.wasm.gc.linear

package tinygogc

import _ "unsafe"

const segmentedHeap = true
const maxHeapSegments = 128

func newHeapSegment(minimum uintptr) (uintptr, uintptr) {
	// Begin with a small heap so explicit GC does not sweep 32 MiB for tiny
	// programs. Double successive arenas up to the former 32 MiB size.
	shift := heapSegmentCount
	if shift > 5 {
		shift = 5
	}
	size := uintptr(1<<20) << shift
	// The nominal arena size includes its metadata. Account for that and
	// alignment at both ends before comparing with the default arena size;
	// even an object slightly smaller than 32 MiB may not fit in that arena.
	extra := minimum/(bytesPerBlock*blocksPerStateByte) + 2*bytesPerBlock
	if minimum > ^uintptr(0)-extra-(wasmPageSize-1) {
		return 0, 0
	}
	if required := alignUp(minimum+extra, wasmPageSize); required > size {
		size = required
	}
	start := gcWasmNewArena(size)
	if start == 0 {
		return 0, 0
	}
	return alignUp(start, bytesPerBlock), alignDown(start+size, bytesPerBlock)
}

//go:linkname gcWasmNewArena C.llgo_gc_new_arena
func gcWasmNewArena(size uintptr) uintptr
