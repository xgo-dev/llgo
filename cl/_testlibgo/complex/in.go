// LITTEST
package main

import (
	"math/cmplx"
)

// CHECK-LABEL: define void @"{{.*}}/cl/_testlibgo/complex.f"({ double, double } %0, { double, double } %1){{.*}} {
func f(c, z complex128) {
	// CHECK: %2 = call double @"math/cmplx.Abs"({ double, double } %0)
	println("abs(3+4i):", cmplx.Abs(c))
	// CHECK: %3 = extractvalue { double, double } %1, 0
	// CHECK: %4 = extractvalue { double, double } %1, 1
	println("real(3+4i):", real(z))
	println("imag(3+4i):", imag(z))
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testlibgo/complex.main"(){{.*}} {
func main() {
	re := 3.0
	im := 4.0
	z := 3 + 4i
	c := complex(re, im)
	// CHECK: call void @"{{.*}}/cl/_testlibgo/complex.f"({ double, double } { double 3.000000e+00, double 4.000000e+00 }, { double, double } { double 3.000000e+00, double 4.000000e+00 })
	f(c, z)
}
