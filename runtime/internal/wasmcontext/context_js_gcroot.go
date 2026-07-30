//go:build llgo && js && wasm && llgo_wasm_gc

package wasmcontext

import (
	"unsafe"

	"github.com/goplus/llgo/runtime/internal/clite/emscripten"
	"github.com/goplus/llgo/runtime/internal/gcroot"
)

func (ctx *Context) Swap(next *Context, nextRoots unsafe.Pointer) {
	gcroot.SwitchAtBoundary((*gcroot.Context)(nextRoots))
	emscripten.FiberSwap(&ctx.fiber, &next.fiber)
}
