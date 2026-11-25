//go:build llgo && wasm

package gotest

import (
	"reflect"
	"testing"
)

//llgo:cdecl
type ffiCallback func(int32) int32

type ffiGoBox struct{ F ffiCallback }

//llgo:cdecl
type ffiNativeBox struct{ F ffiCallback }

//llgo:cdecl
type ffiNativeArray [1]ffiCallback

//llgo:cdecl
type ffiInlineArray struct{ F [1]ffiCallback }

//llgo:cdecl
type ffiNativePair struct {
	F   ffiCallback
	Tag uint32
}

func ffiAdd(x int32) int32 { return x + 7 }

func TestReflectCallNativeCallbackStorage(t *testing.T) {
	f := ffiCallback(ffiAdd)
	call := func(fn, arg any) any {
		return reflect.ValueOf(fn).Call([]reflect.Value{reflect.ValueOf(arg)})[0].Interface()
	}
	check := func(t *testing.T, got ffiCallback) {
		t.Helper()
		if got == nil || got(5) != 12 {
			t.Fatal("callback did not retain its target")
		}
	}
	t.Run("scalar", func(t *testing.T) {
		v := call(func(x ffiCallback) ffiCallback { return x }, f)
		check(t, v.(ffiCallback))
	})
	t.Run("native record", func(t *testing.T) {
		v := call(func(x ffiNativeBox) ffiNativeBox { return x }, ffiNativeBox{f})
		check(t, v.(ffiNativeBox).F)
	})
	t.Run("Go record", func(t *testing.T) {
		v := call(func(x ffiGoBox) ffiGoBox { return x }, ffiGoBox{f})
		check(t, v.(ffiGoBox).F)
	})
	t.Run("Go array", func(t *testing.T) {
		type arr [1]ffiCallback
		v := call(func(x arr) arr { return x }, arr{f})
		check(t, v.(arr)[0])
	})
	t.Run("native array", func(t *testing.T) {
		v := call(func(x ffiNativeArray) ffiNativeArray { return x }, ffiNativeArray{f})
		check(t, v.(ffiNativeArray)[0])
	})
	t.Run("inline C array", func(t *testing.T) {
		v := call(func(x ffiInlineArray) ffiInlineArray { return x }, ffiInlineArray{[1]ffiCallback{f}})
		check(t, v.(ffiInlineArray).F[0])
	})
	t.Run("C record fields", func(t *testing.T) {
		// The next result starts immediately after the narrow C record.
		echo := func(x ffiNativePair) (ffiNativePair, uint32) { return x, 1234567 }
		out := reflect.ValueOf(echo).Call([]reflect.Value{reflect.ValueOf(ffiNativePair{f, 23})})
		v := out[0].Interface().(ffiNativePair)
		check(t, v.F)
		if v.Tag != 23 || out[1].Uint() != 1234567 {
			t.Fatalf("tag, following result = %d, %d; want 23, 1234567", v.Tag, out[1].Uint())
		}
	})
	t.Run("nested", func(t *testing.T) {
		type nest struct{ Inner [1]ffiGoBox }
		v := call(func(x nest) nest { return x }, nest{[1]ffiGoBox{{f}}})
		check(t, v.(nest).Inner[0].F)
	})
	t.Run("multi-field", func(t *testing.T) {
		type pair struct {
			F   ffiCallback
			Tag uint32
		}
		v := call(func(x pair) pair { return x }, pair{f, 23}).(pair)
		check(t, v.F)
		if v.Tag != 23 {
			t.Fatalf("tag = %d, want 23", v.Tag)
		}
	})
}
