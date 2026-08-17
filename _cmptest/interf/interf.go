package main

import (
	"github.com/xgo-dev/llgo/_cmptest/interf/foo"
)

func Foo() any {
	return struct{ v int }{1}
}

func main() {
	v := Foo()
	if x, ok := v.(struct{ v int }); ok {
		println(x.v)
	} else {
		println("Foo: not ok")
	}
	bar := foo.Bar()
	if x, ok := bar.(struct{ V int }); ok {
		println(x.V)
	} else {
		println("Bar: not ok")
	}
	if x, ok := foo.F().(struct{ v int }); ok {
		println(x.v)
	} else {
		println("F: not ok")
	}
}

/* Expected output:
1
1
F: not ok
*/
