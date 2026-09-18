//go:build llgo
// +build llgo

package test

import (
	"reflect"
	"runtime"
	"testing"
	"unsafe"
)

func funcPCABI0(f interface{}) uintptr {
	words := (*[2]unsafe.Pointer)(unsafe.Pointer(&f))
	pc := *(*uintptr)(unsafe.Pointer(words[1]))
	if runtime.GOARCH == "wasm" {
		// Raw wasm function pointers are table indices. The runtime exposes the
		// index with its low two bits reserved for synthetic caller markers.
		pc <<= 2
	}
	return pc
}

func TestFuncPCABI0(t *testing.T) {
	f0 := globalAdd
	f1 := makeNoFree()
	f2 := makeWithFree(5)
	f3 := makeWithFree(7)
	s := &S{v: 1}
	mv := s.Add
	me := (*S).Add

	cases := []struct {
		name string
		fn   interface{}
	}{
		{"global", f0},
		{"no-free", f1},
		{"free-5", f2},
		{"free-7", f3},
		{"method-value", mv},
		{"method-expr", me},
	}

	for _, tc := range cases {
		pc := funcPCABI0(tc.fn)
		if pc == 0 {
			t.Fatalf("%s: FuncPCABI0 returned 0", tc.name)
		}
		if got := reflect.ValueOf(tc.fn).Pointer(); got != pc {
			t.Fatalf("%s: FuncPCABI0 mismatch: got=0x%x want=0x%x", tc.name, pc, got)
		}
	}

	if pc2, pc3 := funcPCABI0(f2), funcPCABI0(f3); pc2 != pc3 {
		t.Fatalf("free closure codeptr should match: pc2=0x%x pc3=0x%x", pc2, pc3)
	}
}
