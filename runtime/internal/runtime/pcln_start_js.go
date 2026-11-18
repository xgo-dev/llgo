//go:build llgo && (llgo_pclntab_required || llgo_pclntab_background) && js && wasm

package runtime

import (
	_ "unsafe"

	"github.com/xgo-dev/llgo/runtime/internal/pcln"
)

func loadWasmPCLNAtStartup() bool {
	return pcln.Preload(func() { pclnTimerSleep(1000000) })
}

//go:linkname pclnTimerSleep time.Sleep
func pclnTimerSleep(ns int64)
