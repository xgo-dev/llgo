// LITTEST
package main

import "reflect"

// SYMBOL-DAG: main{{.*}}Model{{.*}}Escaped

type callbackType string

var callbackTypes = []callbackType{"Original"}

type Model struct{}

//go:noinline
func (Model) Original() string { panic("stale callback name") }

//go:noinline
func (Model) Escaped() string { return "escaped" }

//go:noinline
func replaceThroughPointer(name *callbackType) { *name = "Escaped" }

func main() {
	replaceThroughPointer(&callbackTypes[0])
	v := reflect.ValueOf(Model{})
	for _, name := range callbackTypes {
		println(v.MethodByName(string(name)).Call(nil)[0].String())
	}
}
