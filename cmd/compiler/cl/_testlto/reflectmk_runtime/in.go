package main

import (
	"fmt"
	"reflect"
)

type Point struct {
	x int
	y int
}

func (p *Point) Set(x int, y int) {
	p.x = x
	p.y = y
}

func (p Point) String() string {
	return fmt.Sprintf("(%v,%v)", p.x, p.y)
}

func main() {
	rt := reflect.TypeOf((*Point)(nil)).Elem()
	if t := reflect.ArrayOf(1, rt); t.Elem() != rt {
		panic("arrayOf error")
	}
	if t := reflect.ChanOf(reflect.SendDir, rt); t.Elem() != rt {
		panic("chanOf error")
	}
	if t := reflect.FuncOf([]reflect.Type{rt}, []reflect.Type{rt}, false); t.In(0) != rt || t.Out(0) != rt {
		panic("funcOf error")
	}
	if t := reflect.MapOf(rt, rt); t.Key() != rt || t.Elem() != rt {
		panic("mapOf error")
	}
	if t := reflect.PointerTo(rt); t.Elem() != rt {
		panic("pointerTo error")
	}
	if t := reflect.SliceOf(rt); t.Elem() != rt {
		panic("sliceOf error")
	}
	if t := reflect.StructOf([]reflect.StructField{
		{Name: "T", Type: rt},
	}); t.Field(0).Type != rt {
		panic("structOf error")
	}
	if t := rt.Method(0); t.Name != "String" {
		panic("method error")
	}
	if t, ok := rt.MethodByName("String"); !ok || t.Name != "String" {
		panic("methodByName error")
	}
	v := reflect.ValueOf(&Point{1, 2})
	if r := v.Method(1).Call(nil); r[0].String() != "(1,2)" {
		panic("value.Method error")
	}
	if r := v.MethodByName("String").Call(nil); r[0].String() != "(1,2)" {
		panic("value.MethodByName error")
	}
	method(1)
	methodByName("String")
}

func method(n int) {
	v := reflect.ValueOf(&Point{1, 2})
	if r := v.Method(n).Call(nil); r[0].String() != "(1,2)" {
		panic("value.Method error")
	}
}

func methodByName(name string) {
	v := reflect.ValueOf(&Point{1, 2})
	if r := v.MethodByName(name).Call(nil); r[0].String() != "(1,2)" {
		panic("value.MethodByName error")
	}
}
