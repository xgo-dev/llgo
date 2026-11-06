package runtime

import "unsafe"

// internal/bytealg.MakeNoZero is implemented in the runtime in upstream Go.
// We provide it here via linkname so stdlib packages can call it without
// relying on runtime/internal/lib alt patches.
//
//go:linkname bytealg_MakeNoZero internal/bytealg.MakeNoZero
func bytealg_MakeNoZero(n int) []byte {
	if n < 0 || uintptr(n) > maxAlloc {
		panicmakeslicelen()
	}
	capacity := roundupsize(uintptr(n))
	p := AllocNoScanU(capacity)
	b := unsafe.Slice((*byte)(p), int(capacity))
	return b[:n]
}
