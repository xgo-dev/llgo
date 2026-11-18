//go:build llgo && llgo_pclntab_background && wasm && (js || wasip1)

package runtime

func StartWasmPCLN() {
	go func() { loadWasmPCLNAtStartup() }()
}
