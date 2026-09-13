//go:build llgo && wasm && !llgo.wasm.emscripten.memory64

package ffi

import "unsafe"

type signatureStorage struct {
	cif   Signature
	args  []*Type
	slots []uint32
}

// newSignatureStorage keeps both the typed Go roots and the physical wasm32
// pointer array reachable for the whole lifetime of the embedded CIF. libffi
// retains the physical array address after prep.
func newSignatureStorage(values []*Type) (*Signature, **Type) {
	storage := &signatureStorage{args: append([]*Type(nil), values...)}
	if len(storage.args) == 0 {
		return &storage.cif, nil
	}
	storage.slots = make([]uint32, len(storage.args))
	for i, value := range storage.args {
		storage.slots[i] = uint32(uintptr(unsafe.Pointer(value)))
	}
	return &storage.cif, (**Type)(unsafe.Pointer(&storage.slots[0]))
}

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
