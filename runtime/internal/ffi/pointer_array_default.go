//go:build !llgo || !wasm || llgo.wasm.emscripten.memory64

package ffi

import "unsafe"

func typePointerArray(values []*Type) **Type {
	if len(values) == 0 {
		return nil
	}
	return &values[0]
}

func valuePointerArray(values []unsafe.Pointer) *unsafe.Pointer {
	if len(values) == 0 {
		return nil
	}
	return &values[0]
}
