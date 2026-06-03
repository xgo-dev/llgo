// LITTEST
package apkg

// CHECK-LABEL: define double @"{{.*}}.Max"(double %0, double %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = fcmp ogt double %0, %1
// CHECK-NEXT:   br i1 %2, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:{{.*}}
// CHECK-NEXT:   ret double %0
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:{{.*}}
// CHECK-NEXT:   ret double %1
// CHECK-NEXT: }
func Max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
