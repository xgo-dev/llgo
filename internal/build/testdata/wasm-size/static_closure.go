package main

import "reflect"

var closureOffset = 1
var directClosure = func(v int) int { return v + closureOffset }
var nestedClosure = struct{ Call func(int) int }{func(v int) int { return v * 2 }}
var closureTable = []func(int) int{func(v int) int { return v - 1 }}
var factoryClosure = func() any { return closureObject{42} }
var captureEffects int
var capturedClosure = makeCapturedClosure()

type closureObject struct{ value int }

func (o closureObject) Value() int { return o.value }

func makeCapturedClosure() func() int {
	captureEffects++
	n := 40
	return func() int { n++; return n }
}

func checkStaticClosures() {
	closureOffset = 2
	if directClosure(40) != 42 || nestedClosure.Call(21) != 42 || closureTable[0](43) != 42 {
		panic("static closure value or global access lost")
	}
	if captureEffects != 1 || capturedClosure() != 41 || capturedClosure() != 42 {
		panic("captured closure initialization or environment lost")
	}
	result := reflect.ValueOf(directClosure).Call([]reflect.Value{reflect.ValueOf(40)})
	if result[0].Int() != 42 {
		panic("static closure reflection lost")
	}
	object := factoryClosure()
	if object.(interface{ Value() int }).Value() != 42 ||
		reflect.ValueOf(object).MethodByName("Value").Call(nil)[0].Int() != 42 {
		panic("static closure result method lost")
	}
}
