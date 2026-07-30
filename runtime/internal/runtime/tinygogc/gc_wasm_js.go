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
	if alignment < unsafe.Sizeof(uintptr(0)) || alignment&(alignment-1) != 0 {
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
