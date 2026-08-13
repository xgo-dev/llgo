// LITTEST
package apkg

// CHECK-LABEL: define double @"{{.*}}.Max"(double %0, double %1){{.*}} {
// CHECK: [[MAX_CHOOSE_A:%[0-9]+]] = fcmp ogt double %0, %1
// CHECK-NEXT: br i1 [[MAX_CHOOSE_A]], label %{{.*}}, label %{{.*}}
// CHECK: ret double %0
// CHECK: ret double %1
func Max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
