package main

import (
	"reflect"
	"runtime"
)

type callback func(int) int

func (callback) Marker() {}

//go:noinline
func typeQueries(value any) (string, reflect.Kind, int) {
	t := reflect.TypeOf(value)
	return t.Name(), t.Kind(), t.NumMethod()
}

//go:noinline
func retainedType(value any) reflect.Type {
	t := reflect.TypeOf(value)
	if t.Name() != "callback" {
		panic("query on a returned Type")
	}
	return t
}

//go:noinline
func checkNil() {
	defer func() {
		if _, ok := recover().(runtime.Error); !ok {
			panic("query on a nil Type did not raise a runtime error")
		}
	}()
	_ = reflect.TypeOf(nil).Name()
}

func main() {
	if name, kind, methods := typeQueries(callback(nil)); name != "callback" || kind != reflect.Func || methods != 1 {
		panic("query on a named closure type")
	}
	t := reflect.TypeFor[callback]()
	if retainedType(callback(nil)) != t {
		panic("returned Type lost identity")
	}
	method, ok := t.MethodByName("Marker")
	if !ok || method.Type.In(0) != t {
		panic("returned Type lost method metadata")
	}
	// A method value needs the slow path in Value.Type.
	if reflect.ValueOf(callback(nil)).Method(0).Type() != reflect.TypeFor[func()]() {
		panic("bound method type")
	}
	checkNil()

	dynamic := reflect.StructOf([]reflect.StructField{
		{Name: "Field", Type: reflect.TypeFor[int]()},
	})
	value := reflect.New(dynamic).Elem()
	if name, kind, methods := typeQueries(value.Interface()); name != "" || kind != reflect.Struct || methods != 0 {
		panic("query on a dynamic type")
	}
	if got := value.String(); got != "<struct { Field int } Value>" {
		panic(got)
	}
	println("reflect type queries passed")
}
