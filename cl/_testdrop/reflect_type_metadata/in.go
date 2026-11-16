// LITTEST
package main

import "reflect"

// SYMBOL-DAG: main{{.*}}T{{.*}}Make
// SYMBOL-DAG: main{{.*}}T{{.*}}ZDrop
// SYMBOL-DAG: main{{.*}}U{{.*}}Drop
// SYMBOL-DAG: main{{.*}}U{{.*}}Keep
// SYMBOL-DAG: main{{.*}}V{{.*}}KeepFunc
// SYMBOL-DAG: main{{.*}}W{{.*}}KeepInterface

type T struct{}
type U struct{}
type V struct{}
type W struct{}
type I interface{ hidden() W }

//go:noinline
func (T) Make() U { panic("metadata-only method must not be called") }

//go:noinline
func (T) ZDrop() int { panic("unreachable") }

//go:noinline
func (U) Keep() int { return 42 }

//go:noinline
func (U) Drop() int { panic("unreachable") }

//go:noinline
func (V) KeepFunc() int { return 43 }

//go:noinline
func (W) KeepInterface() int { return 44 }

func main() {
	t := reflect.TypeOf(T{})
	m := t.Method(0)
	println(m.Name)
	byName, ok := t.MethodByName("Make")
	println(ok, byName.Name)
	// The child is only obtained from Method.Type. Its callable method and
	// the parent's callable method slot must both survive.
	u := reflect.New(m.Type.Out(0)).Elem()
	println(u.MethodByName("Keep").Call(nil)[0].Int())
	callable, ok := reflect.TypeOf(V{}).MethodByName("KeepFunc")
	if !ok || !callable.Func.IsValid() {
		panic("observed Func was removed")
	}
	println(callable.Func.Call([]reflect.Value{reflect.ValueOf(V{})})[0].Int())
	// Interface methods expose even unexported signatures, with no concrete
	// implementation or conversion to the interface anywhere in the program.
	im := reflect.TypeOf((*I)(nil)).Elem().Method(0)
	w := reflect.New(im.Type.Out(0)).Elem()
	println(w.MethodByName("KeepInterface").Call(nil)[0].Int())
}
