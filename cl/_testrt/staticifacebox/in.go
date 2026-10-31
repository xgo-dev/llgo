// LITTEST
// Scope: common
package main

type I interface {
	M() int
}

type T int

func (t T) M() int { return int(t) + 1 }

// Constant non-direct iface data (integers) go in a read-only box, not AllocU.
// CHECK-LABEL: define {{.*}} @main.boxed(){{.*}} {
// CHECK-NOT: AllocU
// CHECK: _llgo_ifacebox$
func boxed() any {
	return int64(42)
}

func twoStrings() (any, any) {
	return "aaaaaaaaaaaaaaaaaaaaaaaaaaaa", "bbbbbbbbbbbbbbbbbbbbbbbbbbbb"
}

func main() {
	v := boxed()
	if v.(int64) != 42 {
		panic(v)
	}
	if boxed() != any(int64(42)) {
		panic("eface equality")
	}
	var i I = T(41)
	if i.M() != 42 {
		panic(i.M())
	}
	a, b := twoStrings()
	if a.(string) == b.(string) {
		panic("same-length string boxes collided")
	}
	println("ok")
}
