package ffi_test

import (
	"runtime"
	"testing"
	"unsafe"

	ffi "github.com/xgo-dev/llgo/runtime/internal/ffi"
)

// Keep these tests external: an in-package test variant imports testing, while
// LLGo's runtime implementation imports ffi, forming a test-only package cycle.
// Link directly to the storage constructor so the ownership invariant remains
// covered without pulling a second ffi package variant into the test binary.
//
//go:linkname newSignatureStorage github.com/xgo-dev/llgo/runtime/internal/ffi.newSignatureStorage
func newSignatureStorage(ret *ffi.Type, values []*ffi.Type) (*ffi.Signature, **ffi.Type)

type signatureStorage struct {
	cif  ffi.Signature
	ret  *ffi.Type
	args []*ffi.Type
}

func TestNewSignatureStorageOwnsTypes(t *testing.T) {
	ret := &ffi.Type{}
	first := &ffi.Type{}
	second := &ffi.Type{}
	args := []*ffi.Type{first}
	cif, atype := newSignatureStorage(ret, args)
	args[0] = second

	if got := (*ffi.Type)(ffi.Index((*unsafe.Pointer)(unsafe.Pointer(atype)), 0)); got != first {
		t.Fatalf("libffi argument type = %p, want %p", got, first)
	}
	if got := (*signatureStorage)(unsafe.Pointer(cif)).ret; got != ret {
		t.Fatalf("libffi return type root = %p, want %p", got, ret)
	}
	runtime.KeepAlive(cif)
}

func TestNewAggregateTypeOwnsElementArray(t *testing.T) {
	first := &ffi.Type{}
	second := &ffi.Type{}
	elements := []*ffi.Type{first}
	typ := ffi.StructOf(elements...)
	elements[0] = second

	if got := ffi.TypeElement(typ, 0); got != first {
		t.Fatalf("aggregate element type = %p, want %p", got, first)
	}
	runtime.KeepAlive(typ)
}

func TestTypeElement(t *testing.T) {
	typ := ffi.StructOf(ffi.TypeInt64, ffi.TypeInt8, ffi.TypeInt16)
	for i, want := range []*ffi.Type{ffi.TypeInt64, ffi.TypeInt8, ffi.TypeInt16} {
		if got := ffi.TypeElement(typ, uintptr(i)); got != want {
			t.Fatalf("TypeElement(%d) = %p, want %p", i, got, want)
		}
	}
	if ffi.TypeElement(nil, 0) != nil || ffi.TypeElement(new(ffi.Type), 0) != nil {
		t.Fatal("TypeElement accepted an absent element array")
	}
}

func TestEmptyAggregateElementStorage(t *testing.T) {
	typ := ffi.ArrayOf(ffi.TypeInt32, 0)
	if typ.ByteSize() != 0 || ffi.TypeElement(typ, 0) != ffi.TypeInt8 || ffi.TypeElement(typ, 1) != nil {
		t.Fatal("empty aggregate does not have a native-width terminated element array")
	}
}
