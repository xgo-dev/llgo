//go:build (!llgo || !go1.23) && windows && (amd64 || arm64)

package runtime

import "unsafe"

// Other Go compilers and older toolchains lack structs.Align16. Reserve
// enough storage to select a 16-byte aligned Windows CONTEXT address.
type windowsFaultContextStorage [windowsFaultContextSize + 15]byte

func (storage *windowsFaultContextStorage) context() *windowsFaultContext {
	base := unsafe.Pointer(&storage[0])
	offset := (-uintptr(base)) & 15
	return (*windowsFaultContext)(unsafe.Add(base, offset))
}
