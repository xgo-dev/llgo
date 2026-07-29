//go:build js && wasm && llgo_wasm_gc && llgo.wasm_workers

package tinygogc

import "unsafe"

// Emscripten's pthread support calls the internal musl allocator entry points
// directly when the system allocator is disabled.

//export __libc_malloc
func __libc_malloc(size uintptr) unsafe.Pointer {
	return Alloc(size)
}

//export __libc_calloc
func __libc_calloc(nmemb, size uintptr) unsafe.Pointer {
	return wasmCalloc(nmemb, size)
}

//export __libc_realloc
func __libc_realloc(ptr unsafe.Pointer, size uintptr) unsafe.Pointer {
	return Realloc(ptr, size)
}

//export __libc_free
func __libc_free(ptr unsafe.Pointer) {
}
