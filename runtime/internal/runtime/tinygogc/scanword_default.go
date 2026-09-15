//go:build (baremetal && !nogc) || (wasm && llgo.wasm.gc.linear && llgo.wasm.emscripten.memory64)

package tinygogc

import "unsafe"

const gcScanWordSize = uintptr(unsafe.Sizeof(uintptr(0)))

func loadGCScanWord(addr uintptr) uintptr {
	return *(*uintptr)(unsafe.Pointer(addr))
}
