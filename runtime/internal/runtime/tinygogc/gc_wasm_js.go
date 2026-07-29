//go:build js && wasm && llgo_wasm_gc

package tinygogc

import "unsafe"

func wasmCalloc(nmemb, size uintptr) unsafe.Pointer {
	totalSize := nmemb * size
	if nmemb != 0 && totalSize/nmemb != size {
		return nil
	}
	return Alloc(totalSize)
}

func wasmMemalign(alignment, size uintptr) unsafe.Pointer {
	if !wasmValidMemalign(alignment) {
		return nil
	}
	if alignment <= bytesPerBlock {
		return Alloc(size)
	}
	if size > ^uintptr(0)-(alignment-1) {
		return nil
	}
	return unsafe.Pointer(alignUp(uintptr(Alloc(size+alignment-1)), alignment))
}

func wasmValidMemalign(alignment uintptr) bool {
	return alignment >= unsafe.Sizeof(uintptr(0)) && alignment&(alignment-1) == 0
}

//export malloc
func malloc(size uintptr) unsafe.Pointer {
	return Alloc(size)
}

//export free
func free(ptr unsafe.Pointer) {
}

//export calloc
func calloc(nmemb, size uintptr) unsafe.Pointer {
	return wasmCalloc(nmemb, size)
}

//export realloc
func realloc(ptr unsafe.Pointer, size uintptr) unsafe.Pointer {
	return Realloc(ptr, size)
}

//export memalign
func memalign(alignment, size uintptr) unsafe.Pointer {
	return wasmMemalign(alignment, size)
}

//export posix_memalign
func posix_memalign(result *unsafe.Pointer, alignment, size uintptr) int32 {
	if !wasmValidMemalign(alignment) {
		return 22
	}
	ptr := wasmMemalign(alignment, size)
	if ptr == nil {
		return 12
	}
	*result = ptr
	return 0
}

//export emscripten_builtin_malloc
func emscripten_builtin_malloc(size uintptr) unsafe.Pointer {
	return Alloc(size)
}

//export emscripten_builtin_free
func emscripten_builtin_free(ptr unsafe.Pointer) {
}

//export emscripten_builtin_calloc
func emscripten_builtin_calloc(nmemb, size uintptr) unsafe.Pointer {
	return wasmCalloc(nmemb, size)
}

//export emscripten_builtin_realloc
func emscripten_builtin_realloc(ptr unsafe.Pointer, size uintptr) unsafe.Pointer {
	return Realloc(ptr, size)
}

//export emscripten_builtin_memalign
func emscripten_builtin_memalign(alignment, size uintptr) unsafe.Pointer {
	return wasmMemalign(alignment, size)
}
