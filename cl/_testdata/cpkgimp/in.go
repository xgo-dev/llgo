// LITTEST
package main

import (
	c "github.com/goplus/llgo/cl/_testdata/cpkg"
)

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: [[SUM:%[0-9]+]] = call i64 @add(i64 1, i64 2)
// CHECK-NEXT: [[DOUBLE:%[0-9]+]] = call double @Double(double 3.140000e+00)
// CHECK-NEXT: call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 [[SUM]])
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintFloat"(double [[DOUBLE]])
func main() {
	println(c.Xadd(1, 2), c.Double(3.14))
}
