//go:build ((baremetal && !nogc) || (wasm && llgo.wasm.gc.linear)) && (!llgo || !wasip1 || !llgo.wasi_threads)

package tinygogc

const segmentedHeap = false

func newHeapSegment(minimum uintptr) (uintptr, uintptr) { return 0, 0 }
