//go:build llgo && (llgo_pclntab_required || llgo_pclntab_background) && wasip1 && wasm

package runtime

import "github.com/xgo-dev/llgo/runtime/internal/pcln"

func loadWasmPCLNAtStartup() bool { return pcln.Preload(nil) }
