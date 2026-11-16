// LITTEST
package main

import "reflect"

// SYMBOL-DAG: main{{.*}}Model{{.*}}Replacement
// SYMBOL-DAG: main{{.*}}Model{{.*}}Reassigned

type callbackType string

var callbackTypes = []callbackType{"Original"}

type Model struct{}

//go:noinline
func (Model) Original() string { panic("stale callback name") }

//go:noinline
func (Model) Replacement() string { return "replacement" }

//go:noinline
func (Model) Reassigned() string { return "reassigned" }

//go:noinline
func replaceName() { callbackTypes[0] = "Replacement" }

//go:noinline
func replaceSlice() { callbackTypes = []callbackType{"Reassigned"} }

//go:noinline
func invokeCallbacks() {
	v := reflect.ValueOf(Model{})
	for _, name := range callbackTypes {
		println(v.MethodByName(string(name)).Call(nil)[0].String())
	}
}

func main() {
	replaceName()
	invokeCallbacks()
	replaceSlice()
	invokeCallbacks()
}
