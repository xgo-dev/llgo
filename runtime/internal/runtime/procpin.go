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

// LLGo has no Go P to pin a goroutine to. Keep procPin serialized via procPinMu
// as a fallback for unpatched sync callers that assume exclusive P access (the
// patched sync.Pool in runtime/_patch/sync/pool.go does not call runtime_procPin).
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

// sync/atomic.Value uses procPin in upstream Go to prevent cooperative M:N
// goroutine preemption during its initial type-store CAS and to keep a precise
// collector from observing the temporary &firstStoreInProgress type word.
// On hosted LLGo backends, goroutines run on OS threads without cooperative M:N
// preemption (OS preemption can still occur, just as for concurrent spinners)
// and BDWGC is conservative and never dereferences the type word. A future
// precise/typed native collector must revisit this hook.
//
//go:linkname sync_atomic_runtime_procPin sync/atomic.runtime_procPin
func sync_atomic_runtime_procPin() int {
	return 0
}

//go:linkname sync_atomic_runtime_procUnpin sync/atomic.runtime_procUnpin
func sync_atomic_runtime_procUnpin() {}
