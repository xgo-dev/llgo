//go:build ((baremetal && !nogc) || (wasm && llgo.wasm.gc.linear)) && (!llgo || !wasip1 || !llgo.wasi_threads) && !(llgo && baremetal && tinygo.riscv32 && llgo.idf && !nogc)

package tinygogc

const segmentedHeap = false

func newHeapSegment(minimum uintptr) (uintptr, uintptr) { return 0, 0 }
