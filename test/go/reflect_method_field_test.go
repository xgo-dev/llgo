package gotest

import (
	"reflect"
	"testing"
)

type reflectFieldMethodTarget struct{ Bias int }

func (receiver reflectFieldMethodTarget) Number(n int) int { return n + receiver.Bias }
func (reflectFieldMethodTarget) hidden()                   {}

func TestReflectMethodReachedThroughField(t *testing.T) {
	typ := reflect.TypeOf(struct{ Value reflectFieldMethodTarget }{}).Field(0).Type
	if typ.NumMethod() != 1 {
		t.Fatalf("exported method count = %d, want 1", typ.NumMethod())
	}
	method := typ.Method(0)
	want := reflect.FuncOf([]reflect.Type{typ, reflect.TypeOf(0)}, []reflect.Type{reflect.TypeOf(0)}, false)
	if method.Name != "Number" || method.Type != want || method.Func.Type() != want {
		t.Fatalf("method type = %v, value type = %v, want %v", method.Type, method.Func.Type(), want)
	}
	values := method.Func.Call([]reflect.Value{reflect.ValueOf(reflectFieldMethodTarget{Bias: 7}), reflect.ValueOf(35)})
	if len(values) != 1 || values[0].Int() != 42 {
		t.Fatalf("method results = %v, want [42]", values)
	}
}
