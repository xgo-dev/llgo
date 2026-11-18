//go:build llgo && llgo_pclntab_background && wasm && (wasip1 || (js && !llgo.wasm.workers))

package runtime

func StartWasmPCLN() {
	go func() { loadWasmPCLNAtStartup() }()
}
