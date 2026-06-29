// LITTEST
package main

// CHECK-DAG: call { ptr, i1 } @llvm.type.checked.load(ptr %{{[0-9]+}}, i32 0, metadata !"go.method.A:func() int")
// CHECK-DAG: call { ptr, i1 } @llvm.type.checked.load(ptr %{{[0-9]+}}, i32 0, metadata !"go.method.B:func(int) int")
// CHECK-DAG: !"go.method.A:func() int"
// CHECK-DAG: !"go.method.B:func(int) int"
// SYMBOL-NOT: globaldce_interface_slots{{.*}}T{{.*}}Drop
// SYMBOL-DAG: globaldce_interface_slots{{.*}}T{{.*}}A
// SYMBOL-DAG: globaldce_interface_slots{{.*}}T{{.*}}B
// SYMBOL-NOT: globaldce_interface_slots{{.*}}T{{.*}}Drop

type I interface {
	A() int
	B(int) int
}

type T struct{}

//go:noinline
func (T) A() int {
	return 7
}

//go:noinline
func (T) B(v int) int {
	return v + 5
}

//go:noinline
func (T) Drop() int {
	panic("Drop should be unreachable")
}

func main() {
	var i I = T{}
	println(i.A(), i.B(9))
}
