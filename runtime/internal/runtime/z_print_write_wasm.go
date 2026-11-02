//go:build !llgo || !js || !llgo.wasm.emscripten

// The _wasm.go suffix also restricts this file to GOARCH=wasm.
package runtime

import (
	"unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
)

// PrintString32 accepts a compiler-proven length that fits Wasm32 size_t.
// Scalar arguments avoid a string temporary and a checked word conversion.
func PrintString32(data unsafe.Pointer, size uint32) {
	c.Fwrite(data, 1, uintptr(size), c.Stderr)
}
