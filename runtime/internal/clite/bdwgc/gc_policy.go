//go:build !nogc && !baremetal && !wasm

package bdwgc

import _ "unsafe"

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
