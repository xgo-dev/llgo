//go:build llgo && js && wasm && llgo.wasm_workers && !llgo_wasm_gc

package runtime

import "unsafe"

type wasmWorkerGCState struct{}

func initWasmWorkerGCSystem(*wasmWorker) {}

func wasmWorkerSystemRootPointer(*wasmWorker) unsafe.Pointer { return nil }

func wasmGCRequestPending(*wasmWorker) bool { return false }

func wasmWorkerStopForGC(*wasmWorker) bool { return false }

func wasmGCAllocatorYield() {}
