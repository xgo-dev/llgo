// LITTEST
package main

// CHECK-LABEL: define void @"{{.*}}/cl/_testdata/varinit.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testdata/varinit.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testdata/varinit.init$guard", align 1
// CHECK-NEXT:   store i64 100, ptr @"{{.*}}/cl/_testdata/varinit.a", align 8
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

var a = 100

// CHECK-LABEL: define void @"{{.*}}/cl/_testdata/varinit.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i64, ptr @"{{.*}}/cl/_testdata/varinit.a", align 8
// CHECK-NEXT:   %1 = add i64 %0, 1
// CHECK-NEXT:   store i64 %1, ptr @"{{.*}}/cl/_testdata/varinit.a", align 8
// CHECK-NEXT:   %2 = load i64, ptr @"{{.*}}/cl/_testdata/varinit.a", align 8
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

func main() {
	a++
	_ = a
}
