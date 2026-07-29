//go:build tinygo.wasm

package main

import "unsafe"

func checkWasmModel() {
	if unsafe.Sizeof(uintptr(0)) != 4 {
		panic("-target wasm must use 32-bit words")
	}
	if cLongSize() != 4 {
		panic("-target wasm must use the wasm32 C data model")
	}
}
