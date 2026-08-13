// LITTEST
package main

import (
	"math"
)

// CHECK-LABEL: define void @main.main(){{.*}} {
func main() {
	// Each math result must flow unchanged to the corresponding print.
	// CHECK: [[SQRT:%.*]] = call double @math.Sqrt(double 2.000000e+00)
	// CHECK-NEXT: call void @"{{.*}}PrintFloat"(double [[SQRT]])
	// CHECK: [[ABS:%.*]] = call double @math.Abs(double -1.200000e+00)
	// CHECK-NEXT: call void @"{{.*}}PrintFloat"(double [[ABS]])
	// CHECK: [[LDEXP:%.*]] = call double @math.Ldexp(double 1.200000e+00, i64 3)
	// CHECK-NEXT: call void @"{{.*}}PrintFloat"(double [[LDEXP]])
	println(math.Sqrt(2))
	println(math.Abs(-1.2))
	println(math.Ldexp(1.2, 3))
}
