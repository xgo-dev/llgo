// LITTEST
package main

import (
	"reflect"
)

// SYMBOL-NOT: main{{.*}}Known{{.*}}Drop
// SYMBOL-DAG: main{{.*}}Known{{.*}}Direct
// SYMBOL-DAG: main{{.*}}Known{{.*}}Concat
// SYMBOL-DAG: main{{.*}}Known{{.*}}Slice
// SYMBOL-DAG: main{{.*}}Known{{.*}}Forward
// SYMBOL-NOT: main{{.*}}Known{{.*}}Drop

type Known struct{}

//go:noinline
func (Known) Direct() string { return "direct" }

//go:noinline
func (Known) Concat() string { return "concat" }

//go:noinline
func (Known) Slice() string { return "slice" }

//go:noinline
func (Known) Forward() string { return "forward" }

//go:noinline
func (Known) Drop() string { panic("unreachable") }

func callForward(name string) string {
	out := reflect.ValueOf(Known{}).MethodByName(name).Call(nil)
	return out[0].String()
}

func callConcat(prefix, suffix string) string {
	out := reflect.ValueOf(Known{}).MethodByName(prefix + suffix).Call(nil)
	return out[0].String()
}

func callSlice(source string) string {
	out := reflect.ValueOf(Known{}).MethodByName(source[2:7]).Call(nil)
	return out[0].String()
}

func callUnsignedSlice(source string) string {
	out := reflect.ValueOf(Known{}).MethodByName(source[uint64(2):uint64(7)]).Call(nil)
	return out[0].String()
}

func main() {
	v := reflect.ValueOf(Known{})
	println(v.MethodByName("Direct").Call(nil)[0].String())
	println(callConcat("Con", "cat"))
	println(callSlice("__Slice__"))
	println(callUnsignedSlice("__Slice__"))
	println(callForward("Forward"))
}
