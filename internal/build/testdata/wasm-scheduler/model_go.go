//go:build !tinygo.wasm

package main

import (
	"runtime"
	"unsafe"
)

func checkWasmModel() {
	if runtime.GOOS == "wasip1" {
		if unsafe.Sizeof(uintptr(0)) != 4 {
			panic("GOOS=wasip1 GOARCH=wasm must use 32-bit words")
		}
		if cLongSize() != 4 {
			panic("GOOS=wasip1 GOARCH=wasm must use the wasm32 C data model")
		}
		return
	}
	if unsafe.Sizeof(uintptr(0)) != 8 {
		panic("GOOS/GOARCH wasm must use 64-bit words")
	}
	if cLongSize() != 8 {
		panic("GOOS/GOARCH wasm must use the LP64 C data model")
	}
}
