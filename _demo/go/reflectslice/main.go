package main

import (
	"reflect"
)

type rtype struct {
	flag int
}
type uncommonType struct {
	offset int
}
type method struct {
	Name string
}

func main() {
	demo(2)
}

func demo(count int) {
	tt := reflect.New(reflect.StructOf([]reflect.StructField{
		{Name: "S", Type: reflect.TypeOf(rtype{})},
		{Name: "U", Type: reflect.TypeOf(uncommonType{})},
		{Name: "M", Type: reflect.ArrayOf(count, reflect.TypeOf(method{}))},
	}))
	iface := tt.Elem().Field(2).Slice(0, count).Interface()
	_, ok := iface.([]method)
	if !ok {
		panic("error")
	}
}
