package gotest

import (
	"reflect"
	"testing"
)

type reflectEmptyReceiver struct{}

func (reflectEmptyReceiver) Number(n int) int { return n }

func TestReflectCallZeroSizeArgs(t *testing.T) {
	typ := reflect.TypeOf(struct{ Value reflectEmptyReceiver }{}).Field(0).Type
	method := typ.Method(0)
	tests := []struct {
		name string
		fn   reflect.Value
		args []reflect.Value
	}{
		{"method expression", method.Func, []reflect.Value{reflect.ValueOf(reflectEmptyReceiver{}), reflect.ValueOf(42)}},
		{"method value", reflect.ValueOf(reflectEmptyReceiver{}).Method(0), []reflect.Value{reflect.ValueOf(42)}},
		{"function", reflect.ValueOf(func(_ struct{}, n int, _ [0]int, m int, _ struct{}) int { return n + m }),
			[]reflect.Value{reflect.ValueOf(struct{}{}), reflect.ValueOf(20), reflect.ValueOf([0]int{}), reflect.ValueOf(22), reflect.ValueOf(struct{}{})}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.fn.Call(tt.args)[0].Int(); got != 42 {
				t.Fatalf("Call = %d, want 42", got)
			}
		})
	}
}

func TestReflectMakeFuncZeroSizeArgs(t *testing.T) {
	type signature func(struct{}, int, [0]int, int, struct{}) int
	fn := reflect.MakeFunc(reflect.TypeOf(signature(nil)), func(in []reflect.Value) []reflect.Value {
		if len(in) != 5 || in[0].Type() != reflect.TypeOf(struct{}{}) || in[2].Type() != reflect.TypeOf([0]int{}) || in[4].Type() != reflect.TypeOf(struct{}{}) {
			t.Fatalf("MakeFunc arguments = %v", in)
		}
		return []reflect.Value{reflect.ValueOf(int(in[1].Int() + in[3].Int()))}
	})
	if got := fn.Interface().(signature)(struct{}{}, 20, [0]int{}, 22, struct{}{}); got != 42 {
		t.Fatalf("typed MakeFunc call = %d, want 42", got)
	}
	if got := fn.Call([]reflect.Value{reflect.ValueOf(struct{}{}), reflect.ValueOf(20), reflect.ValueOf([0]int{}), reflect.ValueOf(22), reflect.ValueOf(struct{}{})})[0].Int(); got != 42 {
		t.Fatalf("reflect MakeFunc call = %d, want 42", got)
	}
}

func TestReflectMakeFuncOnlyZeroSizeArgs(t *testing.T) {
	called := false
	type signature func(struct{}, [0]int)
	fn := reflect.MakeFunc(reflect.TypeOf(signature(nil)), func(in []reflect.Value) []reflect.Value {
		called = true
		if len(in) != 2 || in[0].Type() != reflect.TypeOf(struct{}{}) || in[1].Type() != reflect.TypeOf([0]int{}) {
			t.Fatalf("MakeFunc arguments = %v", in)
		}
		return nil
	}).Interface().(signature)
	fn(struct{}{}, [0]int{})
	if !called {
		t.Fatal("MakeFunc callback was not called")
	}
}

func TestReflectMakeFuncZeroSizeArgsMultiResult(t *testing.T) {
	type signature func(struct{}, int, [0]int) (int, int)
	fn := reflect.MakeFunc(reflect.TypeOf(signature(nil)), func(in []reflect.Value) []reflect.Value {
		if len(in) != 3 || in[0].Type() != reflect.TypeOf(struct{}{}) || in[2].Type() != reflect.TypeOf([0]int{}) {
			t.Fatalf("MakeFunc arguments = %v", in)
		}
		return []reflect.Value{in[1], reflect.ValueOf(22)}
	}).Interface().(signature)
	if a, b := fn(struct{}{}, 20, [0]int{}); a != 20 || b != 22 {
		t.Fatalf("MakeFunc results = (%d, %d), want (20, 22)", a, b)
	}
}
