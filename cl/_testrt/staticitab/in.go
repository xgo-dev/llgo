// LITTEST
// Scope: common
package main

type I interface {
	M() int
}

type T int

func (t T) M() int { return int(t) + 1 }

// Known T2I uses a static itab global instead of runtime.NewItab.
// CHECK-LABEL: define {{.*}} @main.boxed({{.*}}){{.*}} {
// CHECK-NOT: NewItab
// CHECK: _llgo_itab$
func boxed(t T) I {
	return t
}

func main() {
	var i I = T(41)
	if i.M() != 42 {
		panic(i.M())
	}
	j := boxed(T(41))
	if i != j {
		panic("iface equality")
	}
	k := any(T(41)).(I)
	if i != k {
		panic("static vs assert itab")
	}
	println("ok")
}
