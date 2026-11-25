// LITTEST
package main

import "reflect"

// SYMBOL-NOT: main{{.*}}T{{.*}}Drop
// SYMBOL-DAG: main{{.*}}T{{.*}}Keep
// SYMBOL-NOT: main{{.*}}T{{.*}}Drop

type T struct {
	n int
}

//go:noinline
func (t T) Keep() int {
	return t.n + 1
}

//go:noinline
func (t T) Drop() int {
	panic("Drop should be unreachable")
}

func main() {
	v := reflect.ValueOf(T{n: 41})
	out := v.MethodByName("Keep").Call(nil)
	println(out[0].Int())
}
