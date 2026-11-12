//go:build llgo && !wasm && !baremetal

package llgoext

import (
	"reflect"
	"testing"
)

//llgo:cdecl
type reflectedNativeCallback func(int) int

func reflectedNativeIncrement(n int) int { return n + 1 }

func TestReflectNativeCallableIdentity(t *testing.T) {
	goValue := reflect.ValueOf(reflectedNativeIncrement)
	native := reflectedNativeCallback(reflectedNativeIncrement)
	nativeValue := reflect.ValueOf(native)
	if goValue.Type().AssignableTo(nativeValue.Type()) || nativeValue.Type().AssignableTo(goValue.Type()) {
		t.Fatal("Go descriptors and native entries must retain their callable roles")
	}
	slot := reflect.New(nativeValue.Type()).Elem()
	slot.Set(nativeValue)
	if result := slot.Call([]reflect.Value{reflect.ValueOf(41)}); len(result) != 1 || result[0].Int() != 42 {
		t.Fatalf("native callback results = %v, want [42]", result)
	}
}
