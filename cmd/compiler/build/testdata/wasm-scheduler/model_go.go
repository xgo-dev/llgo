//go:build !llgo.wasm.emscripten.memory64

package main

import (
	"unsafe"
)

func checkWasmModel() {
	if unsafe.Sizeof(uintptr(0)) != 8 {
		panic("Memory32 profiles must retain the Go 64-bit word model")
	}
	if cLongSize() != 4 {
		panic("Memory32 profiles must use the ILP32 C data model")
	}
}
