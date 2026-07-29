//go:build llgo && js && wasm && llgo.wasm_workers

package wasmevent

import "github.com/goplus/llgo/runtime/internal/clite/pthread/sync"

type runtimeMutex struct {
	mutex sync.Mutex
}

func newRuntimeMutex() runtimeMutex {
	var result runtimeMutex
	if result.mutex.Init(nil) != 0 {
		panic("wasmevent: failed to initialize timer mutex")
	}
	return result
}

func (m *runtimeMutex) Lock() {
	m.mutex.Lock()
}

func (m *runtimeMutex) Unlock() {
	m.mutex.Unlock()
}
