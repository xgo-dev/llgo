//go:build llgo && js && wasm && llgo.wasm.emscripten

package emscripten

import c "github.com/xgo-dev/llgo/runtime/internal/clite"

// WriteStderr bypasses musl stdio, whose write path does not return when it is
// entered from a freshly switched Asyncify fiber. Emscripten implements stdio
// on this synchronous WASI import, which also preserves print's byte stream.
func WriteStderr(data c.Pointer, size uintptr) {
	writeStderr(data, size)
}
