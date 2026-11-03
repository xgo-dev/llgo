//go:build !wasm || (!wasip1 && (!js || llgo.wasm.reflect.libffi))

package gotest

import (
	"reflect"
	"testing"
)

//llgo:cdecl
type reflectNativeMakeFunc func(uintptr) uintptr

func newReflectNativeMakeFunc() reflect.Value {
	delta := uintptr(1)
	return reflect.MakeFunc(reflect.TypeOf(reflectNativeMakeFunc(nil)), func(args []reflect.Value) []reflect.Value {
		return []reflect.Value{reflect.ValueOf(uintptr(args[0].Uint()) + delta)}
	})
}

// Keep these as separate roots: an invalid native entry can terminate its
// process before an ordinary assertion reports the failure.
func TestReflectNativeMakeFuncCall(t *testing.T) {
	value := newReflectNativeMakeFunc()
	out := value.Call([]reflect.Value{reflect.ValueOf(uintptr(41))})
	if len(out) != 1 || out[0].Uint() != 42 {
		t.Fatalf("native MakeFunc.Call returned %v, want one uintptr result of 42", out)
	}
}

func TestReflectNativeMakeFuncInterface(t *testing.T) {
	fn := newReflectNativeMakeFunc().Interface().(reflectNativeMakeFunc)
	if got := fn(41); got != 42 {
		t.Fatalf("native MakeFunc.Interface call = %d, want 42", got)
	}
}
