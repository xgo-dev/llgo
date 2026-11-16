//go:build llgo_pclntab_external && wasm

package runtime

import "github.com/xgo-dev/llgo/runtime/internal/pcln"

func ensureRuntimePCLN() bool { return pcln.Ensure(waitRuntimePCLN) }
func runtimePCLNReady() bool  { return pcln.Ready() }
func waitRuntimePCLN()        { timeSleep(1000000) }
