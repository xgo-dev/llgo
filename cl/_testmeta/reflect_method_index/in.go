package main

import "reflect"

type T struct{}

func (T) M() {}

func useValue() {
	_ = reflect.ValueOf(T{}).Method(0)
}

func useType() {
	_ = reflect.TypeOf(T{}).Method(0)
}

func main() {
	useValue()
	useType()
}
