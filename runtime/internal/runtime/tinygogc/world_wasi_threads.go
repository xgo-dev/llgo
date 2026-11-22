//go:build llgo && wasip1 && wasm && llgo.wasi_threads && llgo.wasm.gc.linear

package tinygogc

import (
	_ "unsafe"

	"github.com/xgo-dev/llgo/runtime/internal/gcroot"
)

func gcStopWorld() bool { return !gcroot.WASIConcurrentMutators() || wasiStopWorld() }

func gcResumeWorld() {
	if gcroot.WASIConcurrentMutators() {
		wasiResumeWorld()
	}
}

//go:linkname wasiStopWorld github.com/xgo-dev/llgo/runtime/internal/runtime.wasiGCStopTheWorld
func wasiStopWorld() bool

//go:linkname wasiResumeWorld github.com/xgo-dev/llgo/runtime/internal/runtime.wasiGCResumeWorld
func wasiResumeWorld()
