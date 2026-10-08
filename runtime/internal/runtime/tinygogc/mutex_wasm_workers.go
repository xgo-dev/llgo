//go:build llgo && js && wasm && llgo.wasm.gc.linear && llgo.wasm.workers

package tinygogc

import (
	_ "unsafe"

	"github.com/xgo-dev/llgo/runtime/internal/wasmsync"
	"github.com/xgo-dev/llgo/runtime/internal/wasmworkers"
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

var gcHandoffWord uint32

func unlockForGC(m *mutex) {
	contended := m.Contended()
	unlock(m)
	if contended {
		// A collector can have no other runnable G on its worker. Gosched then
		// immediately runs it again, so it repeatedly wins the allocator CAS
		// before a woken remote mutator resumes. Give that native worker a
		// bounded scheduling opportunity after resuming the world.
		wasmworkers.Wait(&gcHandoffWord, 0, 1_000_000)
	}
}
