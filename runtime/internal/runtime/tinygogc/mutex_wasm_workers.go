//go:build llgo && js && wasm && llgo_wasm_gc && llgo.wasm_workers

package tinygogc

import (
	_ "unsafe"

	"github.com/goplus/llgo/runtime/internal/wasmsync"
)

type mutex = wasmsync.Mutex

func lock(m *mutex) {
	m.Lock(gcAllocatorYield)
}

func unlock(m *mutex) {
	m.Unlock()
}

//go:linkname gcAllocatorYield github.com/goplus/llgo/runtime/internal/runtime.wasmGCAllocatorYield
func gcAllocatorYield()
