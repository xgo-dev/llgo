//go:build !llgo.wasm.emscripten.memory64

package main

type nativeSignedWord = int32

const (
	expectedGoWordSize   = 8
	expectedCPointerSize = 4
	expectedCLongSize    = 4
)
