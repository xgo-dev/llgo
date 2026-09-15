//go:build !llgo || !wasm || llgo.wasm.emscripten.memory64

package ffi

import "unsafe"

func Index(args *unsafe.Pointer, i uintptr) unsafe.Pointer {
	addr := uintptr(unsafe.Pointer(args)) + i*unsafe.Sizeof(uintptr(0))
	return *(*unsafe.Pointer)(unsafe.Pointer(addr))
}
