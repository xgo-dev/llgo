//go:build ((baremetal && !nogc) || (wasm && llgo.wasm.gc.linear)) && (!llgo || !wasip1 || !wasm || !llgo.wasi_threads || !llgo.wasm.gc.linear)

package tinygogc

const segmentedHeap = false
const maxHeapSegments = 1

func newHeapSegment(minimum uintptr) (uintptr, uintptr) { return 0, 0 }
