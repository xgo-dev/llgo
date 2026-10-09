//go:build wasm && llgo.wasm.gc.linear && (!llgo || !js || !llgo.wasm.workers)

package runtime

import psync "github.com/xgo-dev/llgo/runtime/internal/sync"

type wasmFinalizerMutex = psync.Mutex

var wasmFinalizerOnce psync.Once

func initWasmFinalizers() {
	wasmFinalizers.mu.Init(nil)
	wasmFinalizers.m = make(map[uintptr]*wasmFinalizerEntry)
}

func lockWasmFinalizers() {
	wasmFinalizerOnce.Do(initWasmFinalizers)
	wasmFinalizers.mu.Lock()
}
