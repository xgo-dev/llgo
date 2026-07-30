//go:build llgo && wasip1 && wasm && !llgo.wasi_threads && !llgo_wasm_gc

package wasmcontext

import "unsafe"

func (ctx *Context) Resume(_ unsafe.Pointer) {
	if !ctx.launched {
		contextLaunch(ctx)
		ctx.launched = true
		return
	}
	contextRewind(ctx)
}

func (ctx *Context) Suspend(_ unsafe.Pointer) {
	contextUnwind(ctx)
}

const LLGoFiles = "_asm/context_wasm.S"
