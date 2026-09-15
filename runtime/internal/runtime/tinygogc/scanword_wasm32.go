//go:build wasm && llgo.wasm.gc.linear && !llgo.wasm.emscripten.memory64

package tinygogc

import "unsafe"

// Memory32 hosts may place C pointers at four-byte intervals even though the
// Go language profile stores pointers in eight-byte slots. Scanning every
// physical pointer word finds both layouts: a Go pointer occupies the low word
// of its slot, while C and host-toolchain pointers may occupy either word.
const gcScanWordSize = uintptr(4)

func loadGCScanWord(addr uintptr) uintptr {
	return uintptr(*(*uint32)(unsafe.Pointer(addr)))
}
