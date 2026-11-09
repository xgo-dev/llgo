//go:build !baremetal

package runtime

import "github.com/xgo-dev/llgo/runtime/internal/sync/atomic"

// The marker supplies 8-byte alignment where uint64 has weaker Go alignment.
// Keep the value in our own field so allocator accounting uses runtime intrinsics:
// stdlib method calls may allocate instrumented Wasm caller-location frames.
type memProfileCounter struct {
	_     atomic64Alignment
	value uint64
}

func memProfileAddObject(p *memProfileCounter) {
	atomic.Add(&p.value, uint64(1))
}

func memProfileLoadObjects(p *memProfileCounter) uint64 {
	return atomic.Load(&p.value)
}
