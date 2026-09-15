//go:build llgo.wasm.emscripten.memory64

package main

const (
	expectedGoWordSize   = 8
	expectedCPointerSize = 8
	expectedCLongSize    = 8
)
