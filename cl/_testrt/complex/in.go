// LITTEST
package main

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/complex.main"(){{.*}} {
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintComplex"({ double, double } { double -5.000000e+00, double 1.000000e+01 })
// CHECK: call { double, double } @"{{.*}}/runtime/internal/runtime.Complex128Div"({ double, double } { double 1.000000e+00, double 2.000000e+00 }, { double, double } { double 3.000000e+00, double 4.000000e+00 })
// CHECK: call { double, double } @"{{.*}}/runtime/internal/runtime.Complex128Div"({ double, double } { double 1.000000e+00, double 2.000000e+00 }, { double, double } zeroinitializer)
// CHECK: call { double, double } @"{{.*}}/runtime/internal/runtime.Complex128Div"({ double, double } zeroinitializer, { double, double } zeroinitializer)
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintBool"(i1 true)
type T complex64

func main() {
	a := 1 + 2i
	b := 3 + 4i
	c := 0 + 0i
	println(real(a), imag(a))
	println(-a)
	println(a + b)
	println(a - b)
	println(a * b)
	println(a / b)
	println(a / c)
	println(c / c)
	println(a == a, a != a)
	println(a == b, a != b)
	println(complex128(T(a)) == a)
}
