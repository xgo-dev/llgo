//go:build llgo && wasip1 && wasm && llgo.wasi_threads && llgo.wasm.gc.linear

package sync

import (
	_ "unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
)

//go:linkname wasiGCCondTimedWait C.llgo_wasi_gc_cond_timedwait
func wasiGCCondTimedWait(cond *Cond, mutex *Mutex)

//go:linkname wasiGCSafepoint github.com/xgo-dev/llgo/runtime/internal/runtime.wasiGCSafepoint
func wasiGCSafepoint()

func condWait(cond *Cond, mutex *Mutex) c.Int {
	// All hosted condition waiters must check their predicate after return.
	// A timed wake lets a parked pthread acknowledge a collection. Release
	// the application lock first so another mutator can also reach a safepoint.
	wasiGCCondTimedWait(cond, mutex)
	mutex.Unlock()
	wasiGCSafepoint()
	mutex.Lock()
	return 0
}
