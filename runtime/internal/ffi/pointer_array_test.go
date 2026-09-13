//go:build !llgo

package ffi

import (
	"runtime"
	"testing"
)

func TestNewSignatureStorageOwnsArgumentArray(t *testing.T) {
	first := &Type{}
	second := &Type{}
	args := []*Type{first}
	cif, atype := newSignatureStorage(args)
	args[0] = second

	if got := *atype; got != first {
		t.Fatalf("libffi argument type = %p, want %p", got, first)
	}
	runtime.KeepAlive(cif)
}
