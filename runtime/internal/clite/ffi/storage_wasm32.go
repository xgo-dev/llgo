//go:build llgo && wasm && !llgo.wasm.emscripten.memory64

package ffi

import "unsafe"

// These are C ABI storage slots, not Go pointer or uintptr fields.
type SizeT = uint32
type typeArray = uint32
type typePointer = uint32

func storeTypeArray(p **Type) typeArray   { return uint32(uintptr(unsafe.Pointer(p))) }
func loadTypeArray(p typeArray) **Type    { return (**Type)(unsafe.Pointer(uintptr(p))) }
func loadTypePointer(p typePointer) *Type { return (*Type)(unsafe.Pointer(uintptr(p))) }
