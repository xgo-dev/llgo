// LITTEST
package main

import (
	"reflect"
	"unsafe"
)

// SYMBOL-NOT: main{{.*}}Model{{.*}}Drop
// SYMBOL-DAG: main{{.*}}Model{{.*}}Keep
// SYMBOL-NOT: main{{.*}}Model{{.*}}Drop

type Model struct{ Value int }

//go:noinline
func (m Model) Keep() int { return m.Value + 7 }

//go:noinline
func (Model) Drop() int { panic("metadata-only method must not be called") }

func main() {
	typ := reflect.TypeOf(Model{})
	m, ok := typ.MethodByName("Keep")
	if !ok {
		panic("missing callable method")
	}
	// This executable contains only one unsafe lookup path. A conservative
	// demand from another unsafe case must not mask a missing demand here.
	fn := *(*reflect.Value)(unsafe.Add(unsafe.Pointer(&m.Name), unsafe.Offsetof(m.Func)-unsafe.Offsetof(m.Name)))
	println(fn.Call([]reflect.Value{reflect.ValueOf(Model{Value: 35})})[0].Int())

	// Keep remains callable, while merely observing Drop's metadata must not
	// retain Drop's body. No ordinary call references either method.
	metadata, ok := typ.MethodByName("Drop")
	if !ok {
		panic("missing method metadata")
	}
	println(metadata.Name)
}
