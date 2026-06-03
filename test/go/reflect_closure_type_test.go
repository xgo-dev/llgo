package gotest

import (
	"reflect"
	"testing"
)

func TestReflectClosureParamTypeString(t *testing.T) {
	fn := func(func(int) bool) {}
	typ := reflect.TypeOf(fn)
	if got, want := typ.String(), "func(func(int) bool)"; got != want {
		t.Fatalf("function parameter type = %q, want %q", got, want)
	}
	if got, want := typ.In(0).Kind(), reflect.Func; got != want {
		t.Fatalf("function parameter kind = %v, want %v", got, want)
	}
}

type reflectClosureFieldHolder struct {
	F func(int) bool
}

func TestReflectClosureNestedTypeString(t *testing.T) {
	field := reflect.TypeOf(reflectClosureFieldHolder{}).Field(0).Type
	if got, want := field.String(), "func(int) bool"; got != want {
		t.Fatalf("struct field type = %q, want %q", got, want)
	}
	if got, want := field.Kind(), reflect.Func; got != want {
		t.Fatalf("struct field kind = %v, want %v", got, want)
	}
	elem := reflect.TypeOf([]func(int) bool{}).Elem()
	if got, want := elem.String(), "func(int) bool"; got != want {
		t.Fatalf("slice element type = %q, want %q", got, want)
	}
	if got, want := elem.Kind(), reflect.Func; got != want {
		t.Fatalf("slice element kind = %v, want %v", got, want)
	}
}
