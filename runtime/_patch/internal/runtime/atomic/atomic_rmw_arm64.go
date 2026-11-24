//go:build arm64 && go1.22

package atomic

import "unsafe"

// Go1.22 added the six returning bitwise entries. Keep the exact result width
// instead of reproducing older assembly's full-word writes to uint32 slots.
func And32(ptr *uint32, value uint32) uint32 {
	return atomicAnd32(ptr, value)
}

func Or32(ptr *uint32, value uint32) uint32 {
	return atomicOr32(ptr, value)
}

func And64(ptr *uint64, value uint64) uint64 {
	return atomicAnd64(ptr, value)
}

func Or64(ptr *uint64, value uint64) uint64 {
	return atomicOr64(ptr, value)
}

func Anduintptr(ptr *uintptr, value uintptr) uintptr {
	return uintptr(atomicAnd64((*uint64)(unsafe.Pointer(ptr)), uint64(value)))
}

func Oruintptr(ptr *uintptr, value uintptr) uintptr {
	return uintptr(atomicOr64((*uint64)(unsafe.Pointer(ptr)), uint64(value)))
}
