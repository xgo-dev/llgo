//go:build llgo && !baremetal && !wasm

package runtime

import (
	"unsafe"

	"github.com/xgo-dev/llgo/runtime/internal/clite/bitcast"
)

const memProfileSampling = ^uintptr(0)

// MemProfilePause suppresses allocations made while materializing a profile.
// Without it, rate-1 profiling would observe its own symbolization and buffer
// work and recursively grow the bucket table. The returned value supports
// nested readers and is restored by MemProfileResume.
func MemProfilePause() uintptr {
	state := &memProfileState
	previous := state.remaining
	state.remaining = memProfileSampling
	return previous
}

func MemProfileResume(previous uintptr) {
	memProfileState.remaining = previous
}

// recordMemProfileAlloc is deliberately small enough to inline into AllocZ/U.
// The usual enabled path does one rate load, one TLS address calculation, and
// one countdown update. Random threshold generation and stack capture stay on
// the sampled slow path.
func recordMemProfileAlloc(size uintptr) {
	if size == 0 || MemProfileRatePtr == nil {
		return
	}
	rate := *MemProfileRatePtr
	if rate <= 0 {
		return
	}
	state := &memProfileState
	remaining := state.remaining
	if remaining == memProfileSampling {
		return
	}
	if rate != 1 && size < remaining {
		state.remaining = remaining - size
		return
	}
	recordMemProfileAllocSlow(state, size, rate)
}

//go:noinline
func recordMemProfileAllocSlow(state *memProfileThreadState, size uintptr, rate int) {
	if MemProfileStackCapture == nil {
		return
	}
	next := state.remaining
	if rate != 1 {
		// A zero countdown initializes the stream. A non-zero countdown reached
		// here because this allocation crossed the next sampling point.
		if next == 0 {
			next = memProfileNextThreshold(state, rate)
			if size < next {
				state.remaining = next - size
				return
			}
		}
		next = memProfileNextThreshold(state, rate)
	}
	state.remaining = memProfileSampling
	sampleMemProfileStack(size)
	state.remaining = next
}

func memProfileNextThreshold(state *memProfileThreadState, rate int) uintptr {
	// Exponentially distributed with mean rate, like gc's fastexprand:
	// the memoryless property is required — with any bounded-support
	// distribution a near-periodic allocation pattern phase-locks the
	// sampling points onto the large sites and skews per-site estimates.
	x := state.rand
	if x == 0 {
		x = 0x9e3779b97f4a7c15 ^ uint64(uintptr(unsafe.Pointer(state)))
		if x == 0 {
			x = 0x9e3779b97f4a7c15
		}
	}
	x ^= x >> 12
	x ^= x << 25
	x ^= x >> 27
	state.rand = x
	r := (x * 0x2545f4914f6cdd1d) >> 11 // 53 random bits
	u := float64(r) / (1 << 53)
	if u < 1e-12 {
		u = 1e-12
	}
	t := -lnApprox(u) * float64(rate)
	if t < 1 {
		t = 1
	}
	if max := float64(rate) * 64; t > max {
		t = max
	}
	return uintptr(t)
}

// lnApprox computes ln(u) for u in (0,1] via exponent split and an
// atanh series on the mantissa. Its error is far below sampling noise.
func lnApprox(u float64) float64 {
	const ln2 = 0.6931471805599453
	bits := uint64(bitcast.FromFloat64(u))
	e := int((bits>>52)&0x7ff) - 1023
	mbits := (bits &^ (uint64(0x7ff) << 52)) | (uint64(1023) << 52)
	m := bitcast.ToFloat64(int64(mbits)) // in [1, 2)
	z := (m - 1) / (m + 1)
	z2 := z * z
	lnm := 2 * z * (1 + z2/3 + z2*z2/5 + z2*z2*z2/7)
	return float64(e)*ln2 + lnm
}
