// LITTEST
package main

import (
	"os"
	"reflect"
)

// SYMBOL-DAG: main{{.*}}Model{{.*}}BeforeCreate
// SYMBOL-DAG: main{{.*}}Model{{.*}}Dynamic

type callbackType string

var callbackTypes = []callbackType{"BeforeCreate"}

type Model struct{}

//go:noinline
func (Model) BeforeCreate() string { return "before-create" }

//go:noinline
func (Model) Dynamic() string { return "dynamic" }

//go:noinline
func requestedName() string {
	if len(os.Args) > 1 {
		return os.Args[1]
	}
	return "Dynamic"
}

func main() {
	v := reflect.ValueOf(Model{})
	for _, name := range callbackTypes {
		println(v.MethodByName(string(name)).Call(nil)[0].String())
	}
	// Resolving the earlier callsite must not erase this unknown demand from
	// the same function. Dynamic is absent from callbackTypes.
	println(v.MethodByName(requestedName()).Call(nil)[0].String())
}
