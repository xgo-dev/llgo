//go:build llgo && js && wasm && llgo.wasm.gc.linear && llgo.wasm.workers

package gcroot

import "github.com/xgo-dev/llgo/runtime/internal/wasmsync"

var registryLock wasmsync.Mutex

func lockRegistry() {
	registryLock.Lock(nil)
}

func unlockRegistry() {
	registryLock.Unlock()
}
