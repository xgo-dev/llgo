//go:build llgo && !nogc && !baremetal && js && wasm && llgo.wasm.workers

package runtime

import "unsafe"

func heapProfileG() *g {
	if worker := currentWasmWorker(); worker != nil {
		gp := worker.m.curg
		if gp != nil && gp.context != nil {
			return gp
		}
	}
	return nil
}

func recordHeapProfileWithoutG(unsafe.Pointer, uintptr) {}
