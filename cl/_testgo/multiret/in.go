// LITTEST
package main

var a int = 1

// CHECK-LABEL: define { i64, double } @main.foo(double %0){{.*}} {
// CHECK: [[FOO_INT:%[0-9]+]] = load i64, ptr @main.a
// CHECK-NEXT: [[FOO_PAIR0:%[0-9]+]] = insertvalue { i64, double } undef, i64 [[FOO_INT]], 0
// CHECK-NEXT: [[FOO_PAIR:%[0-9]+]] = insertvalue { i64, double } [[FOO_PAIR0]], double %0, 1
// CHECK-NEXT: ret { i64, double } [[FOO_PAIR]]
func foo(f float64) (int, float64) {
	return a, f
}

func main() {
	// CHECK-LABEL: define void @main.main(){{.*}} {
	// CHECK: [[FOO_RESULT:%[0-9]+]] = call { i64, double } @main.foo(double 2.000000e+00)
	// CHECK-NEXT: [[MAIN_INT:%[0-9]+]] = extractvalue { i64, double } [[FOO_RESULT]], 0
	// CHECK-NEXT: [[MAIN_FLOAT:%[0-9]+]] = extractvalue { i64, double } [[FOO_RESULT]], 1
	// CHECK-NEXT: call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 [[MAIN_INT]])
	// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintFloat"(double [[MAIN_FLOAT]])
	i, f := foo(2.0)
	println(i, f)
}
