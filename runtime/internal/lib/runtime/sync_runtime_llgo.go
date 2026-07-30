//go:build darwin || linux || (llgo && wasip1 && wasm && llgo.wasi_threads)

package runtime

import (
	_ "sync/atomic"
	_ "unsafe"

	psync "github.com/goplus/llgo/runtime/internal/clite/pthread/sync"
)

var poolCleanup func()
var procPinOnce psync.Once
var procPinMu psync.Mutex

func initProcPinMu() {
	procPinMu.Init(nil)
}

//go:linkname sync_runtime_registerPoolCleanup sync.runtime_registerPoolCleanup
func sync_runtime_registerPoolCleanup(cleanup func()) {
	poolCleanup = cleanup
}

//go:linkname sync_runtime_procPin sync.runtime_procPin
func sync_runtime_procPin() int {
	procPinOnce.Do(initProcPinMu)
	procPinMu.Lock()
	return 0
}

//go:linkname sync_runtime_procUnpin sync.runtime_procUnpin
func sync_runtime_procUnpin() {
	procPinMu.Unlock()
}

// sync/atomic.Value expects these package-local runtime hooks.
// In llgo they can share the same no-op pin/unpin behavior as sync.
//
//go:linkname atomic_runtime_procPin sync/atomic.runtime_procPin
func atomic_runtime_procPin() int {
	return sync_runtime_procPin()
}

//go:linkname atomic_runtime_procUnpin sync/atomic.runtime_procUnpin
func atomic_runtime_procUnpin() {
	sync_runtime_procUnpin()
}
