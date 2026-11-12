//go:build !llgo || !wasm || llgo.wasm.emscripten.memory64

package emscripten

import "unsafe"

const expectedFiberSize = uintptr(8) * unsafe.Sizeof(uintptr(0))
