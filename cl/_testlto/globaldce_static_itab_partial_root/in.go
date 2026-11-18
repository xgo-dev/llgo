// LITTEST
package main

import "reflect"

// SYMBOL-DAG: main{{.*}}A{{.*}}M
// SYMBOL-DAG: main{{.*}}B{{.*}}M
// SYMBOL-DAG: main{{.*}}C{{.*}}M
// LTO still emits NewItab for mixed static/dynamic roots; the static template is DCE'd.
// SYMBOL-NOT: _llgo_itab

type I interface {
	M() int
}

type A struct{}
type B struct{}
type C struct{}

func (A) M() int { return 1 }
func (B) M() int { return 2 }
func (C) M() int { return 3 }

//go:noinline
func callStatic(v I) int {
	return v.M()
}

//go:noinline
func callDynamic(v any) int {
	return v.(I).M()
}

//go:noinline
func keepType(v any) bool {
	return reflect.TypeOf(v).Name() == "C"
}

func main() {
	// callStatic is individually resolvable, but callDynamic constructs its
	// itab from a runtime concrete type. The shared M type-id must therefore
	// remain intact, including C.M from the live C descriptor.
	println(callStatic(A{}), callDynamic(B{}), keepType(C{}))
}
