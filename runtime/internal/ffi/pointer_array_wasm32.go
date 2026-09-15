//go:build llgo && wasm && !llgo.wasm.emscripten.memory64

package ffi

import "unsafe"

type signatureStorage struct {
	cif   Signature
	ret   *Type
	args  []*Type
	slots []uint32
}

type aggregateTypeStorage struct {
	typ      Type
	elements []*Type
	slots    []uint32
}

// newSignatureStorage keeps the return type, typed argument roots, and the
// physical wasm32 pointer array reachable for the whole lifetime of the
// embedded CIF. libffi retains their physical addresses after prep.
func newSignatureStorage(ret *Type, values []*Type) (*Signature, **Type) {
	storage := &signatureStorage{ret: ret, args: append([]*Type(nil), values...)}
	if len(storage.args) == 0 {
		return &storage.cif, nil
	}
	storage.slots = make([]uint32, len(storage.args))
	for i, value := range storage.args {
		storage.slots[i] = uint32(uintptr(unsafe.Pointer(value)))
	}
	return &storage.cif, typePointerArrayFromSlots(storage.slots)
}

// newAggregateType keeps typed element roots beside the four-byte pointer
// array consumed by wasm32 libffi. Returning the address of the first field
// keeps the complete storage object reachable for the Type's lifetime.
func newAggregateType(size uintptr, alignment, kind uint16, values []*Type) *Type {
	storage := &aggregateTypeStorage{elements: values}
	if len(storage.elements) != 0 {
		storage.slots = make([]uint32, len(storage.elements))
		for i, value := range storage.elements {
			storage.slots[i] = uint32(uintptr(unsafe.Pointer(value)))
		}
	}
	storage.typ = Type{size, alignment, kind, typePointerArrayFromSlots(storage.slots)}
	return &storage.typ
}

func typePointerArrayFromSlots(slots []uint32) **Type {
	if len(slots) == 0 {
		return nil
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
