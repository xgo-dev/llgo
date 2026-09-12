package main

import "reflect"

type binary func(int64, int64) int64

func add(left, right int64) int64 {
	return left + right
}

func main() {
	typ := reflect.TypeOf(binary(nil))
	args := []reflect.Value{reflect.ValueOf(int64(19)), reflect.ValueOf(int64(23))}
	called := reflect.ValueOf(binary(add)).Call(args)[0].Int()
	made := reflect.MakeFunc(typ, func(in []reflect.Value) []reflect.Value {
		return []reflect.Value{reflect.ValueOf(in[0].Int() + in[1].Int())}
	}).Interface().(binary)(19, 23)
	if called != 42 || made != 42 {
		panic("reflection bridge returned the wrong value")
	}
	println(called, made)
}
