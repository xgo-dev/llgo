// LITTEST
package main

// CHECK-DAG: !"Virtual Function Elim"
// CHECK-DAG: call { ptr, i1 } @llvm.type.checked.load(ptr %{{[0-9]+}}, i32 0, metadata !"go.method.A:func() int")
// CHECK-DAG: call { ptr, i1 } @llvm.type.checked.load(ptr %{{[0-9]+}}, i32 0, metadata !"go.method.B:func(int) int")
// CHECK-DAG: call { ptr, i1 } @llvm.type.checked.load(ptr %{{[0-9]+}}, i32 0, metadata !"go.method.C:func() string")
// CHECK-DAG: @"_llgo_{{.*}}/cl/_testlto/globaldce_interface_matrix.T1" = weak_odr constant {{.*}}, !type !{{[0-9]+}}{{.*}}, !vcall_visibility
// CHECK-DAG: @"*_llgo_{{.*}}/cl/_testlto/globaldce_interface_matrix.T2" = weak_odr constant {{.*}}, !type !{{[0-9]+}}{{.*}}, !vcall_visibility
// CHECK-DAG: !{i64 {{[0-9]+}}, !"go.method.A:func() int"}
// CHECK-DAG: !{i64 {{[0-9]+}}, !"go.method.B:func(int) int"}
// CHECK-DAG: !{i64 {{[0-9]+}}, !"go.method.C:func() string"}
// SYMBOL-NOT: globaldce_interface_matrix{{.*}}T1{{.*}}Drop
// SYMBOL-NOT: globaldce_interface_matrix{{.*}}T2{{.*}}Drop
// SYMBOL-DAG: globaldce_interface_matrix{{.*}}T1{{.*}}A
// SYMBOL-DAG: globaldce_interface_matrix{{.*}}T1{{.*}}B
// SYMBOL-DAG: globaldce_interface_matrix{{.*}}T1{{.*}}C
// SYMBOL-DAG: globaldce_interface_matrix{{.*}}T2{{.*}}A
// SYMBOL-DAG: globaldce_interface_matrix{{.*}}T2{{.*}}B
// SYMBOL-DAG: globaldce_interface_matrix{{.*}}T2{{.*}}C
// SYMBOL-NOT: globaldce_interface_matrix{{.*}}T1{{.*}}Drop
// SYMBOL-NOT: globaldce_interface_matrix{{.*}}T2{{.*}}Drop

type Base interface {
	A() int
}

type Embedded interface {
	Base
	B(int) int
}

type I interface {
	Embedded
	C() string
}

type T1 struct{}

//go:noinline
func (T1) A() int {
	return 11
}

//go:noinline
func (T1) B(v int) int {
	return v + 21
}

//go:noinline
func (T1) C() string {
	return "one"
}

//go:noinline
func (T1) Drop() int {
	panic("T1.Drop should be unreachable")
}

type T2 struct {
	n int
}

//go:noinline
func (t *T2) A() int {
	return t.n
}

//go:noinline
func (t *T2) B(v int) int {
	return v + t.n
}

//go:noinline
func (*T2) C() string {
	return "two"
}

//go:noinline
func (*T2) Drop() int {
	panic("T2.Drop should be unreachable")
}

func use(i I) {
	println(i.A(), i.B(3), i.C())
}

func useBase(b Base) {
	println(b.A())
}

func main() {
	use(T1{})
	use(&T2{n: 12})
	useBase(&T2{n: 15})
}
