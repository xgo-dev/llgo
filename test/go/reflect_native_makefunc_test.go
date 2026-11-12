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

//llgo:cdecl
type reflectNativeMakeFuncArray func([2]uintptr) ([2]uintptr, uintptr)

func TestReflectNativeMakeFuncArray(t *testing.T) {
	input := [2]uintptr{1<<31 + 3, 41}
	want := [2]uintptr{1<<31 + 4, 42}
	value := reflect.MakeFunc(reflect.TypeOf(reflectNativeMakeFuncArray(nil)), func(args []reflect.Value) []reflect.Value {
		got := args[0].Interface().([2]uintptr)
		if got != input {
			t.Fatalf("native array argument = %v, want %v", got, input)
		}
		return []reflect.Value{reflect.ValueOf(want), reflect.ValueOf(want[0])}
	})
	fn := value.Interface().(reflectNativeMakeFuncArray)
	if array, word := fn(input); array != want || word != want[0] {
		t.Fatalf("native array results = (%v, %d), want (%v, %d)", array, word, want, want[0])
	}
	out := value.Call([]reflect.Value{reflect.ValueOf(input)})
	if array, word := out[0].Interface().([2]uintptr), out[1].Uint(); array != want || word != uint64(want[0]) {
		t.Fatalf("reflected native array results = (%v, %d), want (%v, %d)", array, word, want, want[0])
	}
}

//llgo:cdecl
type reflectNativeMakeFuncPointers func([2]*byte) [2]*byte

func TestReflectNativeMakeFuncPointerArray(t *testing.T) {
	a, b := byte(20), byte(22)
	input, want := [2]*byte{&a, &b}, [2]*byte{&b, &a}
	value := reflect.MakeFunc(reflect.TypeOf(reflectNativeMakeFuncPointers(nil)), func(args []reflect.Value) []reflect.Value {
		got := args[0].Interface().([2]*byte)
		if got != input || *got[0]+*got[1] != 42 {
			t.Fatalf("native pointer array argument = %v, want %v", got, input)
		}
		return []reflect.Value{reflect.ValueOf(want)}
	})
	if got := value.Interface().(reflectNativeMakeFuncPointers)(input); got != want {
		t.Fatalf("native pointer array result = %v, want %v", got, want)
	}
	if got := value.Call([]reflect.Value{reflect.ValueOf(input)})[0].Interface().([2]*byte); got != want {
		t.Fatalf("reflected native pointer array result = %v, want %v", got, want)
	}
}
