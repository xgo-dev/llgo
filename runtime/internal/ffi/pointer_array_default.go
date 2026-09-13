//go:build !llgo || !wasm || llgo.wasm.emscripten.memory64

package ffi

import "unsafe"

type signatureStorage struct {
	cif  Signature
	args []*Type
}

// newSignatureStorage keeps the copied argument-type array reachable for the
// whole lifetime of the embedded CIF. libffi retains its address after prep.
func newSignatureStorage(values []*Type) (*Signature, **Type) {
	storage := &signatureStorage{args: append([]*Type(nil), values...)}
	return &storage.cif, typePointerArray(storage.args)
}

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
