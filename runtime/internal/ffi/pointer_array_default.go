//go:build !llgo || !wasm || llgo.wasm.emscripten.memory64

package ffi

import "unsafe"

type signatureStorage struct {
	cif  Signature
	ret  *Type
	args []*Type
}

type aggregateTypeStorage struct {
	typ      Type
	elements []*Type
}

// newSignatureStorage keeps the return type and copied argument-type array
// reachable for the whole lifetime of the embedded CIF. libffi retains their
// addresses after prep.
func newSignatureStorage(ret *Type, values []*Type) (*Signature, **Type) {
	storage := &signatureStorage{ret: ret, args: append([]*Type(nil), values...)}
	return &storage.cif, typePointerArray(storage.args)
}

func newAggregateType(size uintptr, alignment, kind uint16, values []*Type) *Type {
	storage := &aggregateTypeStorage{elements: values}
	storage.typ = Type{size, alignment, kind, typePointerArray(storage.elements)}
	return &storage.typ
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
