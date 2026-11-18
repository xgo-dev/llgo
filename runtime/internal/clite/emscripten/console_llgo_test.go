//go:build llgo && js && wasm && llgo.wasm.emscripten

package emscripten_test

import (
	"testing"
	"unsafe"

	"github.com/xgo-dev/llgo/runtime/internal/clite/emscripten"
)

func TestConsoleHostLayoutWrite(t *testing.T) {
	// The host fd_write must decode both slots correctly; the test runner also
	// captures this distinctive line when validating the Emscripten artifact.
	data := []byte("hostlayout console ABI ok\n")
	emscripten.WriteStderr(unsafe.Pointer(&data[0]), uintptr(len(data)))
	emscripten.WriteStderr(nil, 0)
}
