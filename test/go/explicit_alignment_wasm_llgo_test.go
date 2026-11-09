//go:build llgo && !baremetal && wasm && (wasip1 || (js && !llgo.wasm.reflect.libffi))

package gotest

import (
	"reflect"
	"testing"
)

func TestExplicitAlignmentReflectionCall(t *testing.T) {
	n := 37
	want := aligned64{Value: 29, Pointer: &n}
	// The generated Wasm bridges call lowered Go entries directly, without
	// libffi's restriction on over-aligned argument and result storage.
	identity := reflect.ValueOf(explicitAlignedIdentity)
	got := identity.Call([]reflect.Value{reflect.ValueOf(want)})[0].Interface().(aligned64)
	if got != want {
		t.Fatal("aligned reflected call")
	}
	made := reflect.MakeFunc(identity.Type(), func(in []reflect.Value) []reflect.Value { return in }).Interface().(func(aligned64) aligned64)
	if got := made(want); got != want {
		t.Fatal("aligned reflected function")
	}
}
