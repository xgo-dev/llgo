//go:build llgo && wasip1 && wasm && llgo.wasi_threads && llgo.wasm.gc.linear

package tinygogc

import (
	"unsafe"

	"github.com/xgo-dev/llgo/runtime/internal/sync"
)

// wasi-libc's PTHREAD_MUTEX_INITIALIZER is all zero, so the global allocator
// lock is usable before Go package initialization. The lock publishes roots
// while waiting in C, letting the collector stop allocation waiters.
type mutex = sync.Mutex

func lock(m *mutex) { wasiAllocatorLock(unsafe.Pointer(m)) }

func unlock(m *mutex) { m.Unlock() }

// An explicit GC must let a pending allocator acquire the mutex before it
// returns, so a tight runtime.GC loop cannot starve allocating goroutines.
func unlockForGC(m *mutex) { wasiAllocatorFinish(unsafe.Pointer(m)) }

//go:linkname wasiAllocatorLock github.com/xgo-dev/llgo/runtime/internal/runtime.wasiGCAllocatorLock
func wasiAllocatorLock(m unsafe.Pointer)

//go:linkname wasiAllocatorFinish github.com/xgo-dev/llgo/runtime/internal/runtime.wasiGCAllocatorFinish
func wasiAllocatorFinish(m unsafe.Pointer)
