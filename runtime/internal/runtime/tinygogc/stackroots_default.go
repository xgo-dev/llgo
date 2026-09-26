//go:build wasm && llgo.wasm.gc.linear && (!llgo || !js || !llgo.wasm.workers)

package tinygogc

func gcMarkStackRoots() {
	sp := uintptr(getsp())
	top := gcWasmStackTop()
	if sp < top {
		markRoots(sp, top)
	}
}
