// LITTEST
package main

// CHECK-LINE: @0 = private unnamed_addr constant [5 x i8] c"hello", align 1

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testgo/allocinloop.Foo"(%"{{.*}}/runtime/internal/runtime.String" %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = extractvalue %"{{.*}}/runtime/internal/runtime.String" %0, 1
// CHECK-NEXT:   ret i64 %1
// CHECK-NEXT: }

func Foo(s string) int {
	return len(s)
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/allocinloop.Test"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_2, %_llgo_0
// CHECK-NEXT:   %0 = phi i64 [ 0, %_llgo_0 ], [ %4, %_llgo_2 ]
// CHECK-NEXT:   %1 = phi i64 [ 0, %_llgo_0 ], [ %5, %_llgo_2 ]
// CHECK-NEXT:   %2 = icmp slt i64 %1, 10000000
// CHECK-NEXT:   br i1 %2, label %_llgo_2, label %_llgo_3
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1
// CHECK-NEXT:   %3 = call i64 @"{{.*}}/cl/_testgo/allocinloop.Foo"(%"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 5 })
// CHECK-NEXT:   %4 = add i64 %0, %3
// CHECK-NEXT:   %5 = add i64 %1, 1
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %0)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

func Test() {
	j := 0
	for i := 0; i < 10000000; i++ {
		j += Foo("hello")
	}
	println(j)
}

func main() {
	Test()
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/allocinloop.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testgo/allocinloop.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testgo/allocinloop.init$guard", align 1
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/allocinloop.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/allocinloop.Test"()
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
