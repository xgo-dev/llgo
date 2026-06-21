package main

import (
	"reflect"
	"unsafe"
)

func demo(n int, s string) (bool, int) {
	return true, n + len(s)
}

func main() {
	v1 := reflect.ValueOf(demo)
	check(v1, "demo")

	fn := func(n int, s string) (bool, int) {
		return true, n + len(s)
	}
	v2 := reflect.ValueOf(fn)
	check(v2, "fn")

	nv1 := reflect.New(v1.Type()).Elem()
	nv1.Set(v1)
	check(nv1, "reflect.New demo")

	nv2 := reflect.New(v2.Type()).Elem()
	nv2.Set(v2)
	check(nv2, "reflect.New closure")

	_demo := demo
	nv3 := reflect.NewAt(v1.Type(), unsafe.Pointer(&_demo)).Elem()
	check(nv3, "reflect.NewAt demo")

	nv4 := reflect.NewAt(v2.Type(), unsafe.Pointer(&fn)).Elem()
	check(nv4, "reflect.NewAt closure")
}

func check(v reflect.Value, s string) {
	r := v.Call([]reflect.Value{reflect.ValueOf(100), reflect.ValueOf("hello")})
	if r[0].Bool() != true {
		panic("error r[0]: " + s)
	}
	if r[1].Int() != 100+5 {
		panic("error r[1]: " + s)
	}
}
