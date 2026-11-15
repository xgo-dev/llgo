package main

import (
	"reflect"
	"runtime"

	"example.com/crossinline/bridge"
	"example.com/crossinline/dep"
)

const offset = 2

var value = bridge.Sum
var method = dep.Number.Add

type reader interface{ Read() uint64 }

//go:noinline
func indirect(fn func(uint64, uint64) uint64, x, y uint64) uint64 { return fn(x, y) }

//go:noinline
func indirectMethod(fn func(dep.Number, uint64) uint64, n dep.Number, x uint64) uint64 {
	return fn(n, x)
}

//go:noinline
func invoke(r reader) uint64 { return r.Read() }

func mustPanic(fn func()) {
	defer func() {
		if recover() == nil {
			panic("missing receiver panic")
		}
	}()
	fn()
}

func main() {
	// Go's Func is opaque; LLGo's replacement has a different field layout.
	var nilFunc *runtime.Func
	if nilFunc.Name() != "" {
		panic("nil runtime function name")
	}
	pc, _, _, ok := runtime.Caller(0)
	if !ok || runtime.FuncForPC(pc).Name() != "main.main" {
		panic("runtime function name")
	}
	var invalid reflect.Value
	if invalid.IsValid() {
		panic("invalid replacement value")
	}
	rv := reflect.ValueOf(new(int)).Elem()
	if !rv.IsValid() || !rv.CanAddr() || !rv.CanSet() {
		panic("replacement value methods")
	}
	if bridge.Sum(dep.Next(), dep.Next()) != 3+offset || dep.Counter != 2 {
		panic("argument evaluation")
	}
	for x := range uint64(100) {
		wantSum := x + 3 + offset
		if bridge.Sum(x, 3) != wantSum || indirect(value, x, 3) != wantSum {
			panic("function value")
		}
		n, b := dep.Split(x)
		if n != x+1 || b != (x > 3) {
			panic("tuple")
		}
		if dep.Kept(x) != x+1 {
			panic("noinline")
		}
		number := dep.Number(x)
		if bridge.Add(number, 3) != wantSum || indirectMethod(method, number, 3) != wantSum {
			panic("method expression")
		}
		bound := number.Add
		if bound(3) != wantSum || number.Kept() != x+1 {
			panic("bound method or noinline")
		}
		box := dep.Box{X: x}
		if box.Value() != x || box.Read() != x || *box.Address() != x || invoke(&box) != x {
			panic("receiver")
		}
	}
	var nilBox *dep.Box
	if nilBox.Read() != 7 || invoke(nilBox) != 7 {
		panic("nil pointer receiver")
	}
	if (dep.Embedded{}).Read() != 7 {
		panic("promoted nil pointer receiver")
	}
	mustPanic(func() { nilBox.Value() })
	mustPanic(func() { nilBox.Address() })
	before := dep.Counter
	mustPanic(func() { nilBox.Call(dep.Next()) })
	if dep.Counter != before+1 {
		panic("imported receiver evaluation order")
	}
	mustPanic(func() { _ = (*nilBox).Ignore })
	var outer *dep.Outer
	mustPanic(func() { outer.Read() })
	var nilReader reader
	mustPanic(func() { invoke(nilReader) })
	println("cross inline ok")
}
