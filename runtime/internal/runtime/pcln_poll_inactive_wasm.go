//go:build llgo && wasm && (!js || (!llgo_pclntab_required && !llgo_pclntab_background))

package runtime

func pollWasmPCLN() bool { return false }
