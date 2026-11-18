//go:build cgo && !wasm

package cgo

/*
#include <stdint.h>
#include <stdlib.h>
static uint64_t *allocation_create(void) { return calloc(1, sizeof(uint64_t)); }
static uint64_t allocation_pointer(const void *p) { return *(const uint64_t *)p; }
static uint64_t allocation_address(uintptr_t p) { return *(const uint64_t *)p; }
static uint64_t allocation_scalar(uint64_t x) { return x + 1; }
struct allocation_pair { uint64_t a,b; };
static struct allocation_pair allocation_struct(uint64_t x) {
    return (struct allocation_pair){ x+1, x&1 };
}
*/
import "C"

import "unsafe"

func allocationCreate() unsafe.Pointer { return unsafe.Pointer(C.allocation_create()) }
func allocationFree(p unsafe.Pointer)  { C.free(p) }

//go:noinline
func allocationPointer(p unsafe.Pointer) uint64 { return uint64(C.allocation_pointer(p)) }

//go:noinline
func allocationAddress(p unsafe.Pointer) uint64 {
	return uint64(C.allocation_address(C.uintptr_t(uintptr(p))))
}

//go:noinline
func allocationScalar(x uint64) uint64 { return uint64(C.allocation_scalar(C.uint64_t(x))) }

//go:noinline
func allocationStruct(x uint64) (uint64, uint64) {
	r := C.allocation_struct(C.uint64_t(x))
	return uint64(r.a), uint64(r.b)
}
