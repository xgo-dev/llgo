//go:build llgo && wasm && !llgo.wasm.emscripten.memory64

package ffi

import "unsafe"

// libffi consumes physical C pointer arrays. J32 and W32 store Go pointers in
// eight-byte slots, so materialize separate four-byte arrays at this boundary.
func typePointerArray(values []*Type) **Type {
	if len(values) == 0 {
		return nil
	}
	slots := make([]uint32, len(values))
	for i, value := range values {
		slots[i] = uint32(uintptr(unsafe.Pointer(value)))
	}
	return (**Type)(unsafe.Pointer(&slots[0]))
}

func valuePointerArray(values []unsafe.Pointer) *unsafe.Pointer {
	if len(values) == 0 {
		return nil
	}
	slots := make([]uint32, len(values))
	for i, value := range values {
		slots[i] = uint32(uintptr(value))
	}
	return (*unsafe.Pointer)(unsafe.Pointer(&slots[0]))
}
