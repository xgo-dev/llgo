//go:build !baremetal && (!llgo || wasm)

package runtime

import "github.com/xgo-dev/llgo/runtime/internal/sync/atomic"

// The marker supplies 8-byte alignment where uint64 has weaker Go alignment.
// Keep the value in our own field so allocator accounting uses runtime intrinsics:
// stdlib method calls may allocate instrumented Wasm caller-location frames.
type memProfileCounter struct {
	_     memProfileAlignment
	value uint64
}

var memProfileCounters [memProfileNumBuckets]memProfileCounter

func memProfileAddBucket(idx int) {
	atomic.Add(&memProfileCounters[idx].value, uint64(1))
}

func memProfileLoadBucket(idx int) uint64 {
	return atomic.Load(&memProfileCounters[idx].value)
}
