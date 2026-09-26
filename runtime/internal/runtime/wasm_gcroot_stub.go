//go:build llgo && wasm && !llgo.wasm.gc.linear

package runtime

import "unsafe"

const wasmGCRootEnabled = false

type wasmGCRootContext struct{}

func registerWasmGCRoot(*wasmGCRootContext, bool) {}

func wasmGCRootPointer(*wasmGCRootContext) unsafe.Pointer { return nil }

func adoptWasmGCRoot(*wasmGCRootContext) {}

func suspendWasmGCRoot(*wasmGCRootContext) {}

func publishWasmGCRoot() {}

func finishWasmGCRootRebuild() {}

func unregisterWasmGCRoot(*wasmGCRootContext) {}
