//go:build llgo.wasm.emscripten.memory64

package main

import "unsafe"

func checkWasmModel() {
	if unsafe.Sizeof(uintptr(0)) != 8 {
		panic("Memory64 profiles must use the Go 64-bit word model")
	}
	if cLongSize() != 8 {
		panic("Memory64 profiles must use the LP64 C data model")
	}
}
