//go:build llgo && llgo_pclntab_required && wasm && (js || wasip1)

package runtime

func StartWasmPCLN() {
	if !loadWasmPCLNAtStartup() {
		println("fatal error: runtime: unable to preload required PCLN metadata")
		exitPanic()
	}
}
