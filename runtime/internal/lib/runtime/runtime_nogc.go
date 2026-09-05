//go:build nogc && (!wasm || !llgo.wasm.gc.linear)

package runtime

import "runtime"

// ReadMemStats reports the zero-valued collector state when allocation is
// intentionally configured without a garbage collector.
func ReadMemStats(m *runtime.MemStats) {
	if m != nil {
		*m = runtime.MemStats{}
	}
}

func GC() {}
