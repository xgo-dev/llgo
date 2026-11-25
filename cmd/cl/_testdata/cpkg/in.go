// LITTEST
package C

// CHECK: {{^}}@llvm.compiler.used = appending global [2 x ptr] [ptr @Double, ptr @add], section "llvm.metadata"{{$}}

// CHECK-LABEL: define double @Double(double %0){{.*}} {
// CHECK: [[DOUBLE_RESULT:%[0-9]+]] = fmul double 2.000000e+00, %0
// CHECK-NEXT: ret double [[DOUBLE_RESULT]]
func Double(x float64) float64 {
	return 2 * x
}

// CHECK-LABEL: define i64 @add(i64 %0, i64 %1){{.*}} {
// CHECK: [[XADD_RESULT:%[0-9]+]] = call i64 @"{{.*}}.add"(i64 %0, i64 %1)
// CHECK-NEXT: ret i64 [[XADD_RESULT]]
func Xadd(a, b int) int {
	return add(a, b)
}

// CHECK-LABEL: define i64 @"{{.*}}.add"(i64 %0, i64 %1){{.*}} {
// CHECK: [[ADD_RESULT:%[0-9]+]] = add i64 %0, %1
// CHECK-NEXT: ret i64 [[ADD_RESULT]]
func add(a, b int) int {
	return a + b
}
