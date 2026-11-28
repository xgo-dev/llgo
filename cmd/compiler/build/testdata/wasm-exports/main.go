package main

import _ "unsafe"

const LLGoFiles = "_wrap/exports.c"

//export llgo_go_add
func publicAdd(a, b int32) int32 { return a + b }

//go:linkname nativeAdd C.llgo_public_add
func nativeAdd(a, b int32) int32

func main() {
	if nativeAdd(19, 23) != 42 {
		panic("exported C/Go boundary failed")
	}
	println("wasm exports ready")
}
