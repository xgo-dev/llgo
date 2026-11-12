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

func TestReflectGoUintptrCalls(t *testing.T) {
	// Keep nonzero high bits on 64-bit Go targets, including Memory32 Wasm.
	// On 32-bit Go, the same test still exercises the target's native word.
	high := uint64(1) << 40
	input := uintptr(high) | 41
	typ := reflect.TypeOf((func(uintptr) uintptr)(nil))
	made := reflect.MakeFunc(typ, func(args []reflect.Value) []reflect.Value {
		return []reflect.Value{reflect.ValueOf(uintptr(args[0].Uint()) + 1)}
	})
	ordinary := reflect.ValueOf(func(n uintptr) uintptr { return n + 1 })
	for _, value := range []reflect.Value{ordinary, made} {
		out := value.Call([]reflect.Value{reflect.ValueOf(input)})
		if got := uintptr(out[0].Uint()); got != input+1 {
			t.Fatalf("reflect.Call = %#x, want %#x", got, input+1)
		}
		if got := value.Interface().(func(uintptr) uintptr)(input); got != input+1 {
			t.Fatalf("typed call = %#x, want %#x", got, input+1)
		}
	}
}
