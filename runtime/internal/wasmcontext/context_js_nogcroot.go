//go:build llgo && js && wasm && !llgo_wasm_gc

package wasmcontext

import (
	"unsafe"

	"github.com/goplus/llgo/runtime/internal/clite/emscripten"
)

func (ctx *Context) Swap(next *Context, _ unsafe.Pointer) {
	emscripten.FiberSwap(&ctx.fiber, &next.fiber)
}
