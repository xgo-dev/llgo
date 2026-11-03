package gotest

import (
	"reflect"
	"testing"
)

func TestReflectGoVariadicMakeFunc(t *testing.T) {
	typ := reflect.TypeOf((func(...uintptr) uintptr)(nil))
	value := reflect.MakeFunc(typ, func(args []reflect.Value) []reflect.Value {
		values := args[0]
		result := uintptr(0)
		for i := 0; i < values.Len(); i++ {
			result += uintptr(values.Index(i).Uint())
		}
		return []reflect.Value{reflect.ValueOf(result)}
	})
	for _, call := range []struct {
		name string
		run  func() []reflect.Value
	}{
		{"Call", func() []reflect.Value {
			return value.Call([]reflect.Value{reflect.ValueOf(uintptr(10)), reflect.ValueOf(uintptr(32))})
		}},
		{"CallSlice", func() []reflect.Value { return value.CallSlice([]reflect.Value{reflect.ValueOf([]uintptr{10, 32})}) }},
	} {
		t.Run(call.name, func(t *testing.T) {
			out := call.run()
			if len(out) != 1 || out[0].Uint() != 42 {
				t.Fatalf("Go variadic MakeFunc returned %v, want 42", out)
			}
		})
	}
}
