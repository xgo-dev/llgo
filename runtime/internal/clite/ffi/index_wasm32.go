//go:build llgo && wasm && !llgo.wasm.emscripten.memory64

package ffi

import "unsafe"

func Index(args *unsafe.Pointer, i uintptr) unsafe.Pointer {
	// args is a C void ** supplied by libffi. Its elements follow the physical
	// wasm32 pointer width, not the eight-byte Go pointer storage width.
	addr := uintptr(unsafe.Pointer(args)) + i*4
	return unsafe.Pointer(uintptr(*(*uint32)(unsafe.Pointer(addr))))
}
