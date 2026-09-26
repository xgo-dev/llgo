//go:build llgo && js && wasm && llgo.wasm.workers && !llgo.wasm.gc.linear

package runtime

import "unsafe"

type wasmWorkerGCState struct{}

func initWasmWorkerGCSystem(*wasmWorker) {}

func suspendWasmWorkerGCSystem(*wasmWorker) {}

func resumeWasmWorkerGCSystem(*wasmWorker) {}

func wasmWorkerSystemRootPointer(*wasmWorker) unsafe.Pointer { return nil }

func wasmGCRequestPending(*wasmWorker) bool { return false }

func wasmGCOwnedBy(*wasmWorker) bool { return false }

func wasmWorkerStopForGC(*wasmWorker) bool { return false }

func wasmGCAllocatorYield() {}
