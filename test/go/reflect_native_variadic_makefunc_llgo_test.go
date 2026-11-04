//go:build llgo && (!wasm || (!wasip1 && (!js || llgo.wasm.reflect.libffi)))

package gotest

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

//llgo:cdecl
type reflectNativeVariadicMakeFunc func(fixed uintptr, __llgo_va_list ...any) uintptr

func TestReflectNativeVariadicMakeFuncRequiresFixedPrototype(t *testing.T) {
	defer func() {
		message := fmt.Sprint(recover())
		if !strings.Contains(message, "native") || !strings.Contains(message, "variadic") || !strings.Contains(message, "fixed prototype") {
			t.Fatalf("native variadic MakeFunc panic = %q, want explicit unsupported variadic diagnostic", message)
		}
	}()
	_ = reflect.MakeFunc(reflect.TypeOf(reflectNativeVariadicMakeFunc(nil)), func([]reflect.Value) []reflect.Value {
		return []reflect.Value{reflect.ValueOf(uintptr(42))}
	})
	t.Fatal("MakeFunc accepted a C variadic callback without a fixed callback prototype")
}

// A native variadic signature without the C ellipsis marker still has no
// runtime metadata that proves a fixed callback ABI. Reject that provider
// shape as well, rather than inferring an ABI from its final slice.
//
//llgo:cdecl
type reflectNativeSliceVariadicMakeFunc func(values ...uintptr) uintptr

func TestReflectNativeSliceVariadicMakeFuncRequiresFixedPrototype(t *testing.T) {
	defer func() {
		message := fmt.Sprint(recover())
		if !strings.Contains(message, "native") || !strings.Contains(message, "variadic") || !strings.Contains(message, "fixed prototype") {
			t.Fatalf("native slice-variadic MakeFunc panic = %q, want explicit fixed-prototype diagnostic", message)
		}
	}()
	_ = reflect.MakeFunc(reflect.TypeOf(reflectNativeSliceVariadicMakeFunc(nil)), func([]reflect.Value) []reflect.Value {
		return []reflect.Value{reflect.ValueOf(uintptr(42))}
	})
	t.Fatal("MakeFunc accepted a native variadic callback without a proved fixed prototype")
}
