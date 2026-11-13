//go:build llgo.wasm.emscripten.memory64

package main

type nativeSignedWord = int64

const (
	expectedGoWordSize   = 8
	expectedCPointerSize = 8
	expectedCLongSize    = 8
)
