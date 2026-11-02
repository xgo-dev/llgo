package main

import (
	"os"
	"reflect"
)

type F32 float32
type F64 float64
type C64 complex64
type C128 complex128

func main() {
	var value any
	switch os.Args[1] {
	case "float32":
		value = F32(1.25)
	case "float64":
		value = F64(1.25)
	case "complex64":
		value = C64(1.25 + 2.5i)
	case "complex128":
		value = C128(1.25 + 2.5i)
	default:
		panic("unknown probe")
	}
	if len(os.Args) > 2 {
		reflected := reflect.New(reflect.TypeOf(value)).Elem()
		reflected.Set(reflect.ValueOf(value))
		value = reflected.Interface()
	}
	panic(value)
}
