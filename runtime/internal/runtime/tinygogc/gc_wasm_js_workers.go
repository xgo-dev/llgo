//go:build js && wasm && llgo.wasm.gc.linear && llgo.wasm.workers

package tinygogc

import "unsafe"

// Emscripten pthread support calls the hidden musl allocator entry points
// directly when the system allocator is disabled.

//export __libc_malloc
func libcMalloc(size uintptr) unsafe.Pointer {
	return Alloc(size)
}

//export __libc_realloc
func libcRealloc(ptr unsafe.Pointer, size uintptr) unsafe.Pointer {
	return Realloc(ptr, size)
}

//export __libc_free
func libcFree(ptr unsafe.Pointer) {
}
