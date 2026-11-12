//go:build llgo

package llgoext

import (
	"testing"
	_ "unsafe" // go:linkname
)

//llgo:cdecl
type nativeUintptrCallback func(uintptr) uintptr

func nativeCallbackIncrement(n uintptr) uintptr { return n + 1 }

//go:linkname callNativeUintptrCallback C.llgo_call_uintptr_callback
func callNativeUintptrCallback(callback nativeUintptrCallback, n uintptr) uintptr

//go:linkname nativeUintptrMax C.llgo_native_uintptr_max
func nativeUintptrMax() uintptr

func TestNativeUintptrCallback(t *testing.T) {
	callback := nativeUintptrCallback(nativeCallbackIncrement)
	if got := callback(41); got != 42 {
		t.Fatalf("native callback = %d, want 42", got)
	}
	if got := callNativeUintptrCallback(callback, 41); got != 42 {
		t.Fatalf("callback invoked by C = %d, want 42", got)
	}
	if got := callNativeUintptrCallback(nativeCallbackIncrement, 41); got != 42 {
		t.Fatalf("implicit callback conversion = %d, want 42", got)
	}
	// The reverse conversion also needs a physical ABI adapter on Memory32.
	goFunction := (func(uintptr) uintptr)(callback)
	if got := goFunction(41); got != 42 {
		t.Fatalf("native-to-Go funcval = %d, want 42", got)
	}
	maximum := nativeUintptrMax()
	if got := callNativeUintptrCallback(callback, maximum-1); got != maximum {
		t.Fatalf("callback at native word boundary = %d, want %d", got, maximum)
	}
	if maximum < ^uintptr(0) {
		for _, tt := range []struct {
			name string
			call func()
		}{
			{"native argument", func() { callback(maximum + 1) }},
			{"native result", func() { callback(maximum) }},
			{"Go funcval argument", func() { goFunction(maximum + 1) }},
		} {
			t.Run(tt.name, func(t *testing.T) {
				defer func() {
					if recover() == nil {
						t.Fatal("out-of-range native word conversion did not panic")
					}
				}()
				tt.call()
			})
		}
	}
	var nilGoFunction func(uintptr) uintptr
	if nativeUintptrCallback(nilGoFunction) != nil {
		t.Fatal("nil Go funcval produced a non-nil native callback")
	}
}
