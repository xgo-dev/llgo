package main

import "reflect"

type T struct{}

func (T) M() {}

func use(name string) {
	_ = reflect.ValueOf(T{}).MethodByName(name)
}

func main() {
	use("M")
}
