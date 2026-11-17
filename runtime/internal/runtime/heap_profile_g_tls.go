//go:build llgo && !baremetal && !nogc && (!wasm || (wasip1 && llgo.wasi_threads))

package runtime

import "unsafe"

func heapProfileG() *g {
	return (*g)(unsafe.Pointer(currentG))
}

// Bootstrap main/foreign-thread state only on the cold allocation path. The
// existing exact counters above still count this allocation exactly once.
//
//go:noinline
func recordHeapProfileWithoutG(ptr unsafe.Pointer, size uintptr) {
	if MemProfileRate > 0 && !initializingG {
		recordHeapProfileAlloc(getg(), ptr, size)
	}
}
