package main

import "reflect"

type T struct{}

func (T) M() {}
func (T) m() {}

func main() {
	_, _ = reflect.TypeOf(T{}).MethodByName("M")
	_, _ = reflect.TypeOf(T{}).MethodByName("m")
}
