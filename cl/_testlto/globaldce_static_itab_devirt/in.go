// LITTEST
package main

import "reflect"

// SYMBOL-DAG: main{{.*}}A{{.*}}M
// SYMBOL-NOT: main{{.*}}B{{.*}}M
// After LTO, callM is proven to use A only, so the static itab is unused and dropped.
// SYMBOL-NOT: _llgo_itab

type I interface {
	M() int
}

type A struct{}
type B struct{}

//go:noinline
func (A) M() int {
	return 7
}

//go:noinline
func (B) M() int {
	panic("B.M should be unreachable")
}

// Keep the interface value across a function boundary so the LTO plugin must
// prove that every caller supplies the same static itab.
//
//go:noinline
func callM(v I) int {
	return v.M()
}

//go:noinline
func keepType(v any) bool {
	return reflect.TypeOf(v).Name() == "B"
}

//go:noinline
func reflectedI(v A) I {
	return reflect.ValueOf(v).Interface().(I)
}

func main() {
	// Keep B's type descriptor live without creating a B-to-I conversion. A
	// signature-wide method capability would retain B.M through that descriptor.
	ok := keepType(B{})
	direct := I(A{})
	// LTO T2I still calls NewItab; the static itab is only a de-virt template
	// and is dropped after the plugin. Equality with a reflection-built
	// interface still holds via the (inter, _type) pair if tab pointers differ.
	println(callM(direct), direct == reflectedI(A{}), ok)
}
