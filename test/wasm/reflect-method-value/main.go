package main

import "reflect"

type value int

func (v value) Add(x int) int { return int(v) + x }

type pointer struct{ n int }

func (p *pointer) Add(x int) int { return p.n + x }

type adder interface{ Add(int) int }

func methodValues() (results [5]int) {
	// Keep this executable independent of Call/MakeFunc tests: those entry
	// points would enable bridges and mask a missing method-value root.
	for i, receiver := range []reflect.Value{
		reflect.ValueOf(value(40)),
		reflect.ValueOf(&pointer{40}),
	} {
		results[2*i] = receiver.Method(0).Interface().(func(int) int)(2)
		results[2*i+1] = receiver.MethodByName("Add").Interface().(func(int) int)(2)
	}
	var v adder = value(40)
	results[4] = reflect.ValueOf(&v).Elem().Method(0).Interface().(func(int) int)(2)
	return
}

func main() {
	results := methodValues()
	println("wasm reflect method values:", results[0], results[1], results[2], results[3], results[4])
}
