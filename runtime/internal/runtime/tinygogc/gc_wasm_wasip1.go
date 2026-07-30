//go:build wasip1 && wasm && llgo_wasm_gc

package tinygogc

import "unsafe"

const LLGoPackage = "link: -Wl,--wrap=malloc -Wl,--wrap=free -Wl,--wrap=realloc -Wl,--wrap=calloc"

//export __wrap_malloc
func __wrap_malloc(size uintptr) unsafe.Pointer {
	return Alloc(size)
}

//export __wrap_free
func __wrap_free(ptr unsafe.Pointer) {
}

//export __wrap_calloc
func __wrap_calloc(nmemb, size uintptr) unsafe.Pointer {
	totalSize := nmemb * size
	if nmemb != 0 && totalSize/nmemb != size {
		return nil
	}
	return Alloc(totalSize)
}

//export __wrap_realloc
func __wrap_realloc(ptr unsafe.Pointer, size uintptr) unsafe.Pointer {
	return Realloc(ptr, size)
}
