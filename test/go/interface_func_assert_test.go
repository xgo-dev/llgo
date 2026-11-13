package gotest

import (
	"reflect"
	"testing"
)

type assertFuncA func(int32) int32
type assertFuncB func(int32) int32
type assertFuncAlias = assertFuncA

//llgo:cdecl
type assertNativeFunc func(int32) int32

func assertsFuncType[T any](v any) bool {
	_, ok := v.(T)
	return ok
}

func TestFunctionTypeAssertionIdentity(t *testing.T) {
	plain := func(x int32) int32 { return x + 1 }
	named := assertFuncA(plain)
	native := assertNativeFunc(func(x int32) int32 { return x + 1 })
	for _, tc := range []struct {
		name   string
		value  any
		assert func(any) bool
		want   bool
	}{
		{"named to unnamed", named, assertsFuncType[func(int32) int32], false},
		{"native to unnamed", native, assertsFuncType[func(int32) int32], false},
		{"unnamed to named", plain, assertsFuncType[assertFuncA], false},
		{"distinct named", named, assertsFuncType[assertFuncB], false},
		{"native to Go", native, assertsFuncType[assertFuncA], false},
		{"exact named", named, assertsFuncType[assertFuncA], true},
		{"alias", named, assertsFuncType[assertFuncAlias], true},
		{"exact unnamed", plain, assertsFuncType[func(int32) int32], true},
		{"typed nil named", assertFuncA(nil), assertsFuncType[assertFuncA], true},
		{"typed nil mismatch", assertFuncA(nil), assertsFuncType[func(int32) int32], false},
		{"nil interface", nil, assertsFuncType[func(int32) int32], false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.assert(tc.value); got != tc.want {
				t.Fatalf("assertion of %T = %v, want %v", tc.value, got, tc.want)
			}
		})
	}
	if got := any(native).(assertNativeFunc)(5); got != 6 {
		t.Fatalf("native function round trip returned %d, want 6", got)
	}
	if got := any(assertNativeFunc(nil)).(assertNativeFunc); got != nil {
		t.Fatal("typed nil native function did not round trip")
	}
}

func TestNamedFunctionAssertionPanics(t *testing.T) {
	var value any = assertFuncA(func(x int32) int32 { return x })
	expectPanicContaining(t, "gotest.assertFuncA, not func(int32) int32", func() {
		_ = value.(func(int32) int32)
	})
	expectPanicContaining(t, "gotest.assertFuncA, not gotest.assertNativeFunc", func() {
		_ = value.(assertNativeFunc)
	})
}

func TestFunctionAssertionReflectRoundTrip(t *testing.T) {
	for _, typ := range []reflect.Type{
		reflect.TypeOf((func(int32) int32)(nil)),
		reflect.TypeOf(assertFuncA(nil)),
	} {
		value := reflect.MakeFunc(typ, func(args []reflect.Value) []reflect.Value {
			return []reflect.Value{reflect.ValueOf(int32(args[0].Int()) + 7)}
		}).Interface()
		if typ.Name() == "" {
			fn, ok := value.(func(int32) int32)
			if !ok || fn(5) != 12 {
				t.Fatal("unnamed reflect function assertion or invocation failed")
			}
		} else {
			fn, ok := value.(assertFuncA)
			if !ok || fn(5) != 12 {
				t.Fatal("named reflect function assertion or invocation failed")
			}
			if _, ok := value.(func(int32) int32); ok {
				t.Fatal("named reflect function asserted as an unnamed function")
			}
		}
	}
}
