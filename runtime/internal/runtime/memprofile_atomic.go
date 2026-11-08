//go:build !baremetal

package runtime

import (
	stdatomic "sync/atomic"

	"github.com/xgo-dev/llgo/runtime/internal/sync/atomic"
)

// Uint64 supplies 8-byte alignment even on 32-bit targets. Keep the actual
// value in our own field so allocator accounting can use runtime intrinsics:
// stdlib method calls may allocate instrumented Wasm caller-location frames.
type memProfileCounter struct {
	_     [0]stdatomic.Uint64
	value uint64
}

func memProfileAddObject(p *memProfileCounter) {
	atomic.Add(&p.value, uint64(1))
}

func memProfileLoadObjects(p *memProfileCounter) uint64 {
	return atomic.Load(&p.value)
}
