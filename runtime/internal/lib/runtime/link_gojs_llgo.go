//go:build llgo && js && wasm && !llgo.wasm.emscripten

package runtime

import (
	"unsafe"
	_ "unsafe"
)

//go:linkname goJSRandomData C.llgo_js_random_data
//go:noescape
func goJSRandomData(buffer unsafe.Pointer, length uintptr)

//go:linkname runtimeGetRandomData runtime.getRandomData
func runtimeGetRandomData(buffer []byte) {
	for len(buffer) > 0 {
		n := len(buffer)
		// Web Crypto limits one getRandomValues call to 65536 bytes.
		if n > 65536 {
			n = 65536
		}
		goJSRandomData(unsafe.Pointer(&buffer[0]), uintptr(n))
		buffer = buffer[n:]
	}
}
