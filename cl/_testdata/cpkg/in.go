// LITTEST
package C

// CHECK-LINE: @llvm.compiler.used = appending global [2 x ptr] [ptr @Double, ptr @add], section "llvm.metadata"

// CHECK-LABEL: define double @Double(double %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = fmul double 2.000000e+00, %0
// CHECK-NEXT:   ret double %1
// CHECK-NEXT: }
func Double(x float64) float64 {
	return 2 * x
}

// CHECK-LABEL: define i64 @add(i64 %0, i64 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = call i64 @"{{.*}}.add"(i64 %0, i64 %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }
func Xadd(a, b int) int {
	return add(a, b)
}

// CHECK-LABEL: define i64 @"{{.*}}.add"(i64 %0, i64 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = add i64 %0, %1
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }
func add(a, b int) int {
	return a + b
}
