package main

import "reflect"

type V = reflect.Value

type lookup interface{ MethodByName(string) V }

type value int

func (v value) Add(x int) int { return int(v) + x }

// An unrelated implementation returns an ordinary function Value. Compiler
// reflection assumptions must belong to reflect.Value's wrapper, not this call.
type ordinary struct{}

func (ordinary) MethodByName(string) V {
	return reflect.ValueOf(func(x int) int { return 40 + x })
}

//go:noinline
func invoke(v lookup) int {
	return v.MethodByName("Add").Interface().(func(int) int)(2)
}

func main() {
	// Keep this independent of Method(index), whose broader demand would mask
	// missing MethodByName retention.
	if invoke(reflect.ValueOf(value(40))) != 42 || invoke(ordinary{}) != 42 {
		panic("aliased named method result")
	}
	println("wasm named method aliases: 42 42")
}
