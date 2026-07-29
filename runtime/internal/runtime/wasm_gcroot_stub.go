//go:build llgo && wasm && !llgo_wasm_gc

package runtime

import "unsafe"

const wasmGCRootEnabled = false

type wasmGCRootContext struct{}

func registerWasmGCRoot(*wasmGCRootContext, bool) {}

func wasmGCRootPointer(*wasmGCRootContext) unsafe.Pointer { return nil }

func adoptWasmGCRoot(*wasmGCRootContext) {}

func publishWasmGCRoot() {}

func unregisterWasmGCRoot(*wasmGCRootContext) {}
