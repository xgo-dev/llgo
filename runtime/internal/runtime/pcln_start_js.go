//go:build llgo && (llgo_pclntab_required || llgo_pclntab_background) && js && wasm

package runtime

import "github.com/xgo-dev/llgo/runtime/internal/pcln"

type wasmPCLNWait struct {
	gp    *g
	ready func() bool
}

// The initial goroutine and its background loader run on worker zero.
// Its system scheduler polls this wait until the JS promise completes.
var wasmPCLNStartupWait wasmPCLNWait

func loadWasmPCLNAtStartup() bool { return pcln.Preload(waitWasmPCLN) }

func waitWasmPCLN(ready func() bool) {
	if ready() {
		return
	}
	wasmPCLNStartupWait = wasmPCLNWait{gp: getg(), ready: ready}
	gopark()
}

func pollWasmPCLN() bool {
	wait := wasmPCLNStartupWait
	if wait.gp == nil {
		return false
	}
	if !wait.ready() {
		return true
	}
	wasmPCLNStartupWait = wasmPCLNWait{}
	goready(wait.gp)
	return false
}
