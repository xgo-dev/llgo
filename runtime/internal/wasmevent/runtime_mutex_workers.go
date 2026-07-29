//go:build llgo && js && wasm && llgo.wasm_workers

package wasmevent

import "github.com/goplus/llgo/runtime/internal/wasmsync"

type runtimeMutex struct {
	mutex wasmsync.Mutex
}

var cooperativeSafepoint func()

func newRuntimeMutex() runtimeMutex { return runtimeMutex{} }

func (m *runtimeMutex) Lock() {
	m.mutex.Lock(cooperativeSafepoint)
}

func (m *runtimeMutex) Unlock() {
	m.mutex.Unlock()
}

// InstallCooperativeSafepoint installs the worker scheduler poll used while
// the timer queue lock is contended.
func InstallCooperativeSafepoint(safepoint func()) {
	cooperativeSafepoint = safepoint
}
