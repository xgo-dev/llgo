// LITTEST
package main

func main() {
	_ = use1()
	_ = use2()
}

func use1() int {
	type T int
	return int(id[T](1))
}

func use2() int {
	type T int
	return int(id[T](2))
}

func id[T ~int](v T) T {
	return v
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/tplocaltype.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testgo/tplocaltype.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testgo/tplocaltype.init$guard", align 1
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/tplocaltype.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call i64 @"{{.*}}/cl/_testgo/tplocaltype.use1"()
// CHECK-NEXT:   %1 = call i64 @"{{.*}}/cl/_testgo/tplocaltype.use2"()
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testgo/tplocaltype.use1"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call i64 @"{{.*}}/cl/_testgo/tplocaltype.id[{{.*}}/cl/_testgo/tplocaltype.T.1.0]"(i64 1)
// CHECK-NEXT:   ret i64 %0
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testgo/tplocaltype.use2"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call i64 @"{{.*}}/cl/_testgo/tplocaltype.id[{{.*}}/cl/_testgo/tplocaltype.T.2.0]"(i64 2)
// CHECK-NEXT:   ret i64 %0
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"{{.*}}/cl/_testgo/tplocaltype.id[{{.*}}/cl/_testgo/tplocaltype.T.1.0]"(i64 %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   ret i64 %0
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"{{.*}}/cl/_testgo/tplocaltype.id[{{.*}}/cl/_testgo/tplocaltype.T.2.0]"(i64 %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   ret i64 %0
// CHECK-NEXT: }
