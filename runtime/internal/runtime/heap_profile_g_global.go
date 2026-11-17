//go:build !nogc && !baremetal && (!llgo || (wasm && !(wasip1 && llgo.wasi_threads) && !llgo.wasm.workers))

package runtime

import "unsafe"

func heapProfileG() *g {
	gp := currentG
	if gp != nil && gp.context != nil {
		return gp
	}
	return nil
}

func recordHeapProfileWithoutG(unsafe.Pointer, uintptr) {}
