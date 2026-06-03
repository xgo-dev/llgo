// LITTEST
package main

var a int = 1

// CHECK-LABEL: define { i64, double } @"{{.*}}multiret.foo"(double %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = load i64, ptr @"{{.*}}multiret.a", align 8
// CHECK-NEXT:   %2 = insertvalue { i64, double } undef, i64 %1, 0
// CHECK-NEXT:   %3 = insertvalue { i64, double } %2, double %0, 1
// CHECK-NEXT:   ret { i64, double } %3
func foo(f float64) (int, float64) {
	return a, f
}

// CHECK-LABEL: define void @"{{.*}}multiret.main"(){{.*}} {
func main() {
	// CHECK: call { i64, double } @"{{.*}}multiret.foo"(double 2.000000e+00)
	// CHECK-NEXT: %1 = extractvalue { i64, double } %0, 0
	// CHECK-NEXT: %2 = extractvalue { i64, double } %0, 1
	// CHECK-NEXT: call void @"{{.*}}PrintInt"(i64 %1)
	// CHECK-NEXT: call void @"{{.*}}PrintByte"(i8 32)
	// CHECK-NEXT: call void @"{{.*}}PrintFloat"(double %2)
	// CHECK-NEXT: call void @"{{.*}}PrintByte"(i8 10)
	// CHECK-NEXT: ret void
	i, f := foo(2.0)
	println(i, f)
}
