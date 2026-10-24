//go:build !llgo || !wasm || baremetal || !llgo.wasm.debugger

package runtime

import "unsafe"

const wasmDebuggerEnabled = false

type wasmCallerFrame struct{}

func SetWasmCallerFrameBase(int, uintptr)           {}
func clearWasmPanicFrameBases(*callerLocationStore) {}
func publishWasmCallerStore(*callerLocationStore)   {}
func beginWasmCallerUpdate() unsafe.Pointer         { return nil }
func endWasmCallerUpdate(unsafe.Pointer)            {}
