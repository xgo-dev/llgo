//go:build llgo && js && wasm && llgo_wasm_gc && llgo.wasm_workers

package gcroot

import "github.com/goplus/llgo/runtime/internal/clite/sync/atomic"

var registryLock uint32

func lockRegistry() {
	for {
		if _, ok := atomic.CompareAndExchange(&registryLock, uint32(0), uint32(1)); ok {
			return
		}
	}
}

func unlockRegistry() {
	atomic.Store(&registryLock, uint32(0))
}
