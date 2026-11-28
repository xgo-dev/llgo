//go:build llgo && js && wasm && llgo.wasm.emscripten

package emscripten

import (
	_ "unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
)

// WriteStderr bypasses musl stdio, whose write path does not return when it is
// entered from a freshly switched Asyncify fiber. Emscripten implements stdio
// on this synchronous WASI import, which also preserves print's byte stream.
func WriteStderr(data c.Pointer, size uintptr) {
	writeStderr(data, size)
}

// WriteStderr32 writes a buffer whose length fits both memory widths.
//
//go:linkname WriteStderr32 C.llgo_console_write
//go:noescape
func WriteStderr32(data c.Pointer, size uint32)

//go:linkname WriteStderrByte C.llgo_console_byte
func WriteStderrByte(data uint8)
