//go:build darwin || linux || windows

package runtime

import (
	_ "unsafe"

	psync "github.com/xgo-dev/llgo/runtime/internal/sync"
)

var procPinOnce psync.Once
var procPinMu psync.Mutex

func initProcPinMu() {
	procPinMu.Init(nil)
}

// LLGo has no Go P to pin a goroutine to. Serialize sync.Pool procPin regions
// so poolLocal access remains mutually exclusive on hosted backends.
func procPin() int {
	procPinOnce.Do(initProcPinMu)
	procPinMu.Lock()
	return 0
}

func procUnpin() {
	procPinMu.Unlock()
}

//go:linkname sync_runtime_procPin sync.runtime_procPin
func sync_runtime_procPin() int {
	return procPin()
}

//go:linkname sync_runtime_procUnpin sync.runtime_procUnpin
func sync_runtime_procUnpin() {
	procUnpin()
}

// sync/atomic.Value uses procPin only to prevent M:N goroutine preemption
// during its initial type-store CAS, never for cross-thread mutual exclusion.
// Hosted LLGo goroutines run on OS threads without cooperative preemption.
//
//go:linkname sync_atomic_runtime_procPin sync/atomic.runtime_procPin
func sync_atomic_runtime_procPin() int {
	return 0
}

//go:linkname sync_atomic_runtime_procUnpin sync/atomic.runtime_procUnpin
func sync_atomic_runtime_procUnpin() {}
