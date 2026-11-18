//go:build !llgo || !wasm || llgo.wasm.emscripten.memory64

package ffi

// SizeT is libffi's native size_t storage. Use Type.ByteSize for Go arithmetic.
type SizeT = uintptr
type typeArray = **Type
type typePointer = *Type

func storeTypeArray(p **Type) typeArray   { return p }
func loadTypeArray(p typeArray) **Type    { return p }
func loadTypePointer(p typePointer) *Type { return p }
