//go:build llgo && js && wasm && llgo.wasm.workers

package js

import llruntime "github.com/xgo-dev/llgo/runtime/internal/runtime"

// The forwarding helpers are separate, non-inlined calls so their captured
// variables are allocated only on remote workers.
func isRemoteJSWorker() bool {
	return llruntime.SchedulerProcID() != 0
}

func onJSWorker(fn func()) {
	if !llruntime.CallMainJSWorker(fn) {
		fn()
	}
}
