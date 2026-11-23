//go:build llgo && js && wasm && llgo.wasm.emscripten

package runtime

import (
	"unsafe"

	"github.com/xgo-dev/llgo/runtime/internal/clite/emscripten"
)

const platformLLGoFiles = ""

func PrintByte(v byte) {
	emscripten.WriteStderrByte(v)
}

func PrintString(s String) {
	emscripten.WriteStderr(s.data, uintptr(s.len))
}

func PrintString32(data unsafe.Pointer, size uint32) {
	emscripten.WriteStderr32(data, size)
}
