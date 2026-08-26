//go:build darwin || linux

package runtime

import (
	_ "unsafe"

	psync "github.com/xgo-dev/llgo/runtime/internal/clite/pthread/sync"
)

var procPinOnce psync.Once
var procPinMu psync.Mutex

func initProcPinMu() {
	procPinMu.Init(nil)
}

// LLGo has no Go P to pin a goroutine to. Serialize procPin regions instead,
// preserving the exclusion that sync.Pool and sync/atomic.Value require.
func procPin() int {
	procPinOnce.Do(initProcPinMu)
	procPinMu.Lock()
	return 0
}

func procUnpin() {
	procPinMu.Unlock()
}

//go:linkname syncRuntimeProcPin sync.runtime_procPin
func syncRuntimeProcPin() int {
	return procPin()
}

//go:linkname syncRuntimeProcUnpin sync.runtime_procUnpin
func syncRuntimeProcUnpin() {
	procUnpin()
}

//go:linkname atomicRuntimeProcPin sync/atomic.runtime_procPin
func atomicRuntimeProcPin() int {
	return procPin()
}

//go:linkname atomicRuntimeProcUnpin sync/atomic.runtime_procUnpin
func atomicRuntimeProcUnpin() {
	procUnpin()
}
