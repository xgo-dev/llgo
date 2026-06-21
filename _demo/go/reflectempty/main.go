package main

import (
	"reflect"
)

var (
	emtpyStruct = reflect.TypeOf((*struct{})(nil)).Elem()
	emtpyArray  = reflect.TypeOf([0]int{})
	tyInt       = reflect.TypeOf(0)
	tyString    = reflect.TypeOf("")
)

func main() {
	ftyp := reflect.FuncOf([]reflect.Type{emtpyStruct, tyInt, emtpyArray, emtpyStruct, tyString}, []reflect.Type{emtpyStruct, tyInt, emtpyArray, tyString}, false)
	fn := reflect.MakeFunc(ftyp, func(args []reflect.Value) []reflect.Value {
		if args[4].String() != "hello world" {
			panic("error")
		}
		return []reflect.Value{args[0], reflect.ValueOf(int(args[1].Int()) + args[4].Len()), args[2], args[4]}
	})
	r := fn.Call([]reflect.Value{reflect.ValueOf(struct{}{}), reflect.ValueOf(100), reflect.ValueOf([0]int{}), reflect.ValueOf(struct{}{}), reflect.ValueOf("hello world")})
	if r[0].Interface() != struct{}{} {
		panic("error r0")
	}
	if r[1].Int() != 111 {
		panic("error r1")
	}
	if r[2].Interface() != [0]int{} {
		panic("error r2")
	}
	if r[3].Interface() != "hello world" {
		panic("error r3")
	}
}
