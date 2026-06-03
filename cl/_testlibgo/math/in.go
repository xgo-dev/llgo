// LITTEST
package main

import (
	"math"
)

// CHECK-LABEL: define void @"{{.*}}/cl/_testlibgo/math.main"(){{.*}} {
func main() {
	// CHECK: %0 = call double @math.Sqrt(double 2.000000e+00)
	// CHECK: %1 = call double @math.Abs(double -1.200000e+00)
	// CHECK: %2 = call double @math.Ldexp(double 1.200000e+00, i64 3)
	println(math.Sqrt(2))
	println(math.Abs(-1.2))
	println(math.Ldexp(1.2, 3))
}
