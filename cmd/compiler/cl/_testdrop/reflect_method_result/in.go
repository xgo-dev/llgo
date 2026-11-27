// LITTEST
package main

import "reflect"

// SYMBOL-NOT: main{{.*}}T{{.*}}Drop
// SYMBOL-NOT: main{{.*}}U{{.*}}Drop
// SYMBOL-DAG: main{{.*}}T{{.*}}Make
// SYMBOL-DAG: main{{.*}}U{{.*}}Keep
// SYMBOL-NOT: main{{.*}}T{{.*}}Drop
// SYMBOL-NOT: main{{.*}}U{{.*}}Drop

type T struct{}
type U struct{}

//go:noinline
func (T) Make() U { return U{} }

//go:noinline
func (T) Drop() U { panic("T.Drop should be unreachable") }

//go:noinline
func (U) Keep() int { return 42 }

//go:noinline
func (U) Drop() int { panic("U.Drop should be unreachable") }

func main() {
	u := reflect.ValueOf(T{}).MethodByName("Make").Call(nil)[0]
	println(u.MethodByName("Keep").Call(nil)[0].Int())
}
