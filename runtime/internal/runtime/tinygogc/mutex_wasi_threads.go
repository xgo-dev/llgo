//go:build llgo && wasip1 && wasm && llgo.wasi_threads && llgo.wasm.gc.linear

package tinygogc

import (
	_ "unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
	"github.com/xgo-dev/llgo/runtime/internal/sync/atomic"
)

type mutex struct{ state uint32 }

func lock(m *mutex) {
	for {
		if _, ok := atomic.CompareAndExchange(&m.state, uint32(0), uint32(1)); ok {
			return
		}
		wasiGCAllocatorYield()
		_ = wasiThreadYield()
	}
}

func unlock(m *mutex) { atomic.Store(&m.state, uint32(0)) }

//go:linkname wasiGCAllocatorYield github.com/xgo-dev/llgo/runtime/internal/runtime.wasiGCSafepoint
func wasiGCAllocatorYield()

//go:linkname wasiThreadYield C.sched_yield
func wasiThreadYield() c.Int
