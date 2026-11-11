// LITTEST
package main

import (
	"reflect"
	"unsafe"
)

// SYMBOL-DAG: main{{.*}}Model{{.*}}Drop
// SYMBOL-DAG: main{{.*}}Model{{.*}}Keep

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
	fn := *(*reflect.Value)(unsafe.Add(unsafe.Pointer(&m), unsafe.Offsetof(m.Func)))
	println(fn.Call([]reflect.Value{reflect.ValueOf(Model{Value: 35})})[0].Int())

	// Both lookups conservatively retain callable slots, including the lookup
	// that only observes metadata. No ordinary call references either method.
	metadata, ok := typ.MethodByName("Drop")
	if !ok {
		panic("missing method metadata")
	}
	println(metadata.Name)
}
