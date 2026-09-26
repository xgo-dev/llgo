//go:build llgo && wasip1 && wasm && llgo.wasi_threads && llgo.wasm.gc.linear

package tinygogc

import _ "unsafe"

const segmentedHeap = true

func newHeapSegment(minimum uintptr) (uintptr, uintptr) {
	const segmentSize = uintptr(32 << 20)
	size := segmentSize
	if minimum > size {
		extra := minimum/(bytesPerBlock*blocksPerStateByte) + bytesPerBlock
		if minimum > ^uintptr(0)-extra-(wasmPageSize-1) {
			return 0, 0
		}
		size = alignUp(minimum+extra, wasmPageSize)
	}
	start := gcWasmNewArena(size)
	if start == 0 {
		return 0, 0
	}
	return alignUp(start, bytesPerBlock), alignDown(start+size, bytesPerBlock)
}

//go:linkname gcWasmNewArena C.llgo_gc_new_arena
func gcWasmNewArena(size uintptr) uintptr
