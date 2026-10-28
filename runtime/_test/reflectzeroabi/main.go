package main

import (
	"fmt"
	"reflect"
	"runtime"
)

type marker struct{}
type emptyArray [0]byte
type emptyReceiver struct{}
type receiver struct{ bias int }

type mixer interface {
	Mix(marker, int, emptyArray, int, marker) int
}

func (r receiver) Mix(_ marker, a int, _ emptyArray, b int, _ marker) int {
	return r.bias + 10*a + b
}

func (emptyReceiver) Mix(_ marker, a int, _ emptyArray, b int, _ marker) int {
	return 10*a + b
}

func (emptyArray) Mix(_ marker, a int, _ emptyArray, b int, _ marker) int {
	return 10*a + b
}

func require(ok bool, message string) {
	if !ok {
		panic(message)
	}
}

func checkCall() {
	tests := []struct {
		fn   any
		args []any
	}{
		{func(_ marker, value int) int { return value + 34 }, []any{marker{}, 7}},
		{func(a int, _ marker, b int) int { return 10*a + b }, []any{4, marker{}, 1}},
		{func(a int, _ marker) int { return a }, []any{41, marker{}}},
		{func(_ emptyArray, a int, _ marker, b int, _ emptyArray) int { return 10*a + b }, []any{emptyArray{}, 4, marker{}, 1, emptyArray{}}},
	}
	for _, test := range tests {
		args := make([]reflect.Value, len(test.args))
		for i, value := range test.args {
			args[i] = reflect.ValueOf(value)
		}
		result := reflect.ValueOf(test.fn).Call(args)
		require(len(result) == 1 && result[0].Int() == 41, "Call shifted a zero-sized argument")
	}
	owner := []int{17}
	fn := func(_ marker, value int, _ emptyArray) int { return owner[0] + value }
	runtime.GC()
	out := reflect.ValueOf(fn).Call([]reflect.Value{reflect.ValueOf(marker{}), reflect.ValueOf(24), reflect.ValueOf(emptyArray{})})
	require(out[0].Int() == 41, "captured closure lost its environment")

	method := reflect.ValueOf(receiver{bias: 10}).MethodByName("Mix")
	args := []reflect.Value{reflect.ValueOf(marker{}), reflect.ValueOf(3), reflect.ValueOf(emptyArray{}), reflect.ValueOf(1), reflect.ValueOf(marker{})}
	require(method.Call(args)[0].Int() == 41, "method Call shifted the receiver or arguments")
	bound := method.Interface().(func(marker, int, emptyArray, int, marker) int)
	require(bound(marker{}, 3, emptyArray{}, 1, marker{}) == 41, "method value wrapper shifted arguments")
}

func checkMakeFunc() {
	type mixed func(marker, int, emptyArray, int, marker) int
	var retained []reflect.Value
	fn := reflect.MakeFunc(reflect.TypeOf(mixed(nil)), func(args []reflect.Value) []reflect.Value {
		require(len(args) == 5, "MakeFunc lost logical zero-sized arguments")
		require(args[0].Type() == reflect.TypeOf(marker{}) && args[2].Type() == reflect.TypeOf(emptyArray{}) && args[4].Type() == reflect.TypeOf(marker{}), "MakeFunc lost zero-sized argument types")
		retained = args
		return []reflect.Value{reflect.ValueOf(int(args[1].Int()*10 + args[3].Int()))}
	}).Interface().(mixed)
	require(fn(marker{}, 4, emptyArray{}, 1, marker{}) == 41, "MakeFunc shifted a nonempty argument")
	runtime.GC()
	require(retained[1].Int() == 4 && retained[3].Int() == 1, "MakeFunc retained native frame storage")
	for _, i := range []int{0, 2, 4} {
		require(retained[i].IsValid() && retained[i].IsZero(), "MakeFunc invalid zero-sized Value")
	}

	called := false
	void := reflect.MakeFunc(reflect.TypeOf((func(marker, emptyArray))(nil)), func(args []reflect.Value) []reflect.Value {
		require(len(args) == 2 && args[0].IsZero() && args[1].IsZero(), "all-empty MakeFunc arguments")
		called = true
		return nil
	}).Interface().(func(marker, emptyArray))
	void(marker{}, emptyArray{})
	require(called, "all-empty MakeFunc was not called")

	pair := reflect.MakeFunc(reflect.TypeOf((func(marker, int, emptyArray) (int, int))(nil)), func(args []reflect.Value) []reflect.Value {
		require(len(args) == 3 && args[0].IsZero() && args[2].IsZero(), "multiple-result argument mapping")
		return []reflect.Value{args[1], reflect.ValueOf(int(args[1].Int() + 1))}
	}).Interface().(func(marker, int, emptyArray) (int, int))
	first, second := pair(marker{}, 40, emptyArray{})
	require(first == 40 && second == 41, "multiple-result MakeFunc shifted arguments")
}

func checkEmptyReceiver() {
	args := []reflect.Value{reflect.ValueOf(marker{}), reflect.ValueOf(4), reflect.ValueOf(emptyArray{}), reflect.ValueOf(1), reflect.ValueOf(marker{})}
	for _, value := range []mixer{emptyReceiver{}, emptyArray{}} {
		// Reflective methods call an interface wrapper with a receiver pointer,
		// even when the source receiver itself has no storage.
		for _, reflected := range []reflect.Value{reflect.ValueOf(value), reflect.ValueOf(&value).Elem()} {
			method := reflected.MethodByName("Mix")
			require(method.Call(args)[0].Int() == 41, "empty receiver method Call shifted arguments")
			bound := method.Interface().(func(marker, int, emptyArray, int, marker) int)
			runtime.GC()
			require(bound(marker{}, 4, emptyArray{}, 1, marker{}) == 41, "empty receiver method Interface shifted arguments")
		}
	}

	// A method expression instead exposes the empty receiver as an ordinary
	// source argument. It must use the zero-sized-argument mapping.
	expressionArgs := append([]reflect.Value{reflect.ValueOf(emptyReceiver{})}, args...)
	require(reflect.ValueOf(emptyReceiver.Mix).Call(expressionArgs)[0].Int() == 41, "empty receiver method expression shifted arguments")
	arrayExpressionArgs := append([]reflect.Value{reflect.ValueOf(emptyArray{})}, args...)
	require(reflect.ValueOf(emptyArray.Mix).Call(arrayExpressionArgs)[0].Int() == 41, "empty array receiver method expression shifted arguments")
}

func main() {
	checkCall()
	checkMakeFunc()
	checkEmptyReceiver()
	fmt.Println("reflect zero-sized argument ABI verified")
}
