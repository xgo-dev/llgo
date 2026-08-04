//go:build !tinygo.wasm

package main

import "unsafe"

func checkWasmModel() {
	if unsafe.Sizeof(uintptr(0)) != 8 {
		panic("GOOS/GOARCH wasm must use 64-bit words")
	}
	if cLongSize() != 8 {
		panic("GOOS/GOARCH wasm must use the LP64 C data model")
	}
}
