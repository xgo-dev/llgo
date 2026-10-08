//go:build llgo && js && wasm && llgo.wasm.workers && llgo.wasm.gc.linear

package runtime

type wasmFinalizerMutex = wasmSemaMutex

func lockWasmFinalizers() {
	// Map updates and collector cancellation can allocate or wait for the GC
	// lock. A contending worker must remain able to acknowledge an STW request
	// instead of blocking in pthread_mutex_lock while the owner is stopped.
	wasmFinalizers.mu.Lock()
	if wasmFinalizers.m == nil {
		wasmFinalizers.m = make(map[uintptr]*wasmFinalizerEntry)
	}
}
