package main

import (
	"reflect"
)

func demo(n int, s string) (bool, int) {
	return true, n + len(s)
}

//llgo:type C
type CFunc func(n int, s string) (bool, int)

type T func(n int, s string) (bool, int)

func (t T) Demo() int {
	return 100
}

func (t T) Call(s string) (bool, int) {
	return t(100, s)
}

func main() {
	v1 := reflect.ValueOf(demo)
	typ := reflect.TypeOf((*T)(nil)).Elem()
	if typ.Kind() != reflect.Func {
		panic("kind error: " + typ.Kind().String())
	}
	if typ.NumIn() != 2 {
		panic("bad num in")
	}
	if typ.NumOut() != 2 {
		panic("bad num out")
	}
	if typ.IsVariadic() {
		panic("not variadic")
	}
	if typ.NumMethod() != 2 {
		panic("error methods")
	}
	v2 := reflect.New(typ).Elem()
	if v2.Type() != typ {
		panic("bad type")
	}
	v2.Set(v1)
	check(v2, "named")

	r := v2.Method(1).Call(nil)
	if r[0].Int() != 100 {
		panic("error call")
	}
	r = v2.MethodByName("Call").Call([]reflect.Value{reflect.ValueOf("hello")})
	if r[0].Bool() != true {
		panic("error r[0]")
	}
	if r[1].Int() != 100+5 {
		panic("error r[1]")
	}

	ctyp := reflect.TypeOf((*CFunc)(nil)).Elem()
	if ctyp.Kind() != reflect.Func {
		panic("kind error: " + ctyp.Kind().String())
	}
	v3 := reflect.New(ctyp).Elem()
	if v3.Type() != ctyp {
		panic("bad c named type")
	}
	v3.Set(v1)
	check(v3, "c named")
}

func check(v reflect.Value, s string) {
	if v.Kind() != reflect.Func {
		panic("error")
	}
	r := v.Call([]reflect.Value{reflect.ValueOf(100), reflect.ValueOf("hello")})
	if r[0].Bool() != true {
		panic("error r[0]: " + s)
	}
	if r[1].Int() != 100+5 {
		panic("error r[1]: " + s)
	}
}
