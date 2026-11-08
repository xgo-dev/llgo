//go:build llgo && js && wasm && llgo.wasm.emscripten && !llgo.wasm.emscripten.memory64

package emscripten_test

import (
	"github.com/xgo-dev/llgo/runtime/internal/clite/emscripten"
	"testing"
)

func TestConsoleRejectsOversizedLength(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("console length was truncated to wasm32")
		}
	}()
	emscripten.WriteStderr(nil, uintptr(1)<<32)
}
