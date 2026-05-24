// LITTEST
package main

var a int = 1

// CHECK-LABEL: define { i64, double } @"{{.*}}/cl/_testgo/multiret.foo"(double %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = load i64, ptr @"{{.*}}/cl/_testgo/multiret.a", align 8
// CHECK-NEXT:   %2 = insertvalue { i64, double } undef, i64 %1, 0
// CHECK-NEXT:   %3 = insertvalue { i64, double } %2, double %0, 1
// CHECK-NEXT:   ret { i64, double } %3
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/multiret.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testgo/multiret.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testgo/multiret.init$guard", align 1
// CHECK-NEXT:   store i64 1, ptr @"{{.*}}/cl/_testgo/multiret.a", align 8
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

func foo(f float64) (int, float64) {
	return a, f
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/multiret.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call { i64, double } @"{{.*}}/cl/_testgo/multiret.foo"(double 2.000000e+00)
// CHECK-NEXT:   %1 = extractvalue { i64, double } %0, 0
// CHECK-NEXT:   %2 = extractvalue { i64, double } %0, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %1)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintFloat"(double %2)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

func main() {
	i, f := foo(2.0)
	println(i, f)
}
