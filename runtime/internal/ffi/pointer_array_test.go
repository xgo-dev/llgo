//go:build !llgo

package ffi

import (
	"runtime"
	"testing"
	"unsafe"
)

func TestNewSignatureStorageOwnsTypes(t *testing.T) {
	ret := &Type{}
	first := &Type{}
	second := &Type{}
	args := []*Type{first}
	cif, atype := newSignatureStorage(ret, args)
	args[0] = second

	if got := *atype; got != first {
		t.Fatalf("libffi argument type = %p, want %p", got, first)
	}
	if got := (*signatureStorage)(unsafe.Pointer(cif)).ret; got != ret {
		t.Fatalf("libffi return type root = %p, want %p", got, ret)
	}
	runtime.KeepAlive(cif)
}

func TestNewAggregateTypeOwnsElementArray(t *testing.T) {
	first := &Type{}
	second := &Type{}
	elements := []*Type{first}
	typ := StructOf(elements...)
	elements[0] = second

	if got := *typ.Elements; got != first {
		t.Fatalf("aggregate element type = %p, want %p", got, first)
	}
	runtime.KeepAlive(typ)
}

func TestTypeElement(t *testing.T) {
	typ := StructOf(TypeInt64, TypeInt8, TypeInt16)
	for i, want := range []*Type{TypeInt64, TypeInt8, TypeInt16} {
		if got := TypeElement(typ, uintptr(i)); got != want {
			t.Fatalf("TypeElement(%d) = %p, want %p", i, got, want)
		}
	}
	if TypeElement(nil, 0) != nil || TypeElement(new(Type), 0) != nil {
		t.Fatal("TypeElement accepted an absent element array")
	}
}
