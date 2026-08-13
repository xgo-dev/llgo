// LITTEST
package main

import (
	"math/cmplx"
)

// CHECK-LABEL: define void @main.f({ double, double } %0, { double, double } %1){{.*}} {
func f(c, z complex128) {
	// CHECK: [[COMPLEX_ABS:%[0-9]+]] = call double @"math/cmplx.Abs"({ double, double } %0)
	// CHECK: call void @"{{.*}}.PrintFloat"(double [[COMPLEX_ABS]])
	println("abs(3+4i):", cmplx.Abs(c))
	// CHECK: [[COMPLEX_REAL:%[0-9]+]] = extractvalue { double, double } %1, 0
	// CHECK: call void @"{{.*}}.PrintFloat"(double [[COMPLEX_REAL]])
	println("real(3+4i):", real(z))
	// CHECK: [[COMPLEX_IMAG:%[0-9]+]] = extractvalue { double, double } %1, 1
	// CHECK: call void @"{{.*}}.PrintFloat"(double [[COMPLEX_IMAG]])
	println("imag(3+4i):", imag(z))
}

// CHECK-LABEL: define void @main.main(){{.*}} {
func main() {
	re := 3.0
	im := 4.0
	z := 3 + 4i
	c := complex(re, im)
	// CHECK: call void @main.f({ double, double } { double 3.000000e+00, double 4.000000e+00 }, { double, double } { double 3.000000e+00, double 4.000000e+00 })
	f(c, z)
}
