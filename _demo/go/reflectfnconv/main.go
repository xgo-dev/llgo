package main

import (
	"reflect"
	"strconv"
)

type itoaFunc func(i int) string

func (f itoaFunc) Itoa(i int) string { return f(i) }

func main() {
	typ := reflect.TypeOf((*itoaFunc)(nil)).Elem()
	tyString := reflect.TypeOf("")
	v := reflect.MakeFunc(typ, func(args []reflect.Value) []reflect.Value {
		r := strconv.Itoa(int(args[0].Int()))
		return []reflect.Value{reflect.ValueOf(r)}
	})
	ftyp := reflect.FuncOf([]reflect.Type{v.Type()}, []reflect.Type{tyString}, false)
	fn := reflect.MakeFunc(ftyp, func(args []reflect.Value) []reflect.Value {
		r := args[0].Call([]reflect.Value{reflect.ValueOf(100)})
		return r
	})
	r := fn.Call([]reflect.Value{v})
	if r[0].String() != "100" {
		panic("func conv error: " + r[0].String())
	}
}
