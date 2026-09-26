//go:build llgo && wasip1 && wasm && llgo.wasi_threads && llgo.wasm.gc.linear

package tinygogc

import "github.com/xgo-dev/llgo/runtime/internal/gcroot"

func gcMarkStackRoots() {
	sp := uintptr(getsp())
	top := gcWasmStackTop()
	if sp < top {
		markRoots(sp, top)
	}
	gcroot.VisitThreadStacks(markRoots)
}
