//go:build !llgo || !wasm || llgo.wasm.emscripten.memory64

package emscripten

type fiberWord = uintptr
