//go:build llgo && js && wasm && llgo.wasm.gc.linear && llgo.wasm.workers

package tinygogc

import (
	_ "unsafe"

	"github.com/xgo-dev/llgo/runtime/internal/wasmsync"
)

type mutex = wasmsync.Mutex

func lock(m *mutex) {
	m.Lock(gcAllocatorYield)
}

func unlock(m *mutex) {
	m.Unlock()
}

//go:linkname gcAllocatorYield github.com/xgo-dev/llgo/runtime/internal/runtime.wasmGCAllocatorYield
func gcAllocatorYield()
