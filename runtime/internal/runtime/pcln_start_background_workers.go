//go:build llgo && llgo_pclntab_background && js && wasm && llgo.wasm.workers

package runtime

func StartWasmPCLN() {
	spawnMainJSCall(func() { loadWasmPCLNAtStartup() })
}
