//go:build !nogc && !baremetal && !wasm

package bdwgc

import (
	_ "unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
)

const gcPolicyLLGoFiles = "; _wrap/gc_policy.c"

// InitPolicy installs the native collector's allocation budget after GC_init.
//
//go:linkname InitPolicy C.llgo_gc_init_policy
func InitPolicy()

// SetGCPercent returns the previous percentage and updates automatic collection.
// Explicit GC_gcollect remains available when the percentage is negative.
//
//go:linkname SetGCPercent C.llgo_gc_set_percent
func SetGCPercent(percent int32) int32

// ZeroAtomicLarge clears a large pointer-free BDWGC allocation, avoiding eager
// page faults for untouched interior pages where supported by the host OS.
//
//go:linkname ZeroAtomicLarge C.llgo_gc_zero_atomic_large
func ZeroAtomicLarge(ptr c.Pointer, size uintptr)
