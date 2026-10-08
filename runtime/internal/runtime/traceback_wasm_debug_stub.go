//go:build !llgo || !wasm || baremetal || !llgo.wasm.debugger

package runtime

import "unsafe"

const wasmDebuggerEnabled = false

func publishWasmCallerStore(*callerLocationStore) {}
func beginWasmCallerUpdate() unsafe.Pointer       { return nil }
func endWasmCallerUpdate(unsafe.Pointer)          {}
