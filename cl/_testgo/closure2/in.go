// LITTEST
package main

// CHECK-LINE: @0 = private unnamed_addr constant [7 x i8] c"closure", align 1

func main() {
	x := 1
	f := func(i int) func(int) {
		return func(i int) {
			println("closure", i, x)
		}
	}
	f(1)(2)
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/closure2.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testgo/closure2.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testgo/closure2.init$guard", align 1
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/closure2.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   store i64 1, ptr %0, align 8
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   %2 = getelementptr inbounds { ptr }, ptr %1, i32 0, i32 0
// CHECK-NEXT:   store ptr %0, ptr %2, align 8
// CHECK-NEXT:   %3 = insertvalue { ptr, ptr } { ptr @"{{.*}}/cl/_testgo/closure2.main$1", ptr undef }, ptr %1, 1
// CHECK-NEXT:   %4 = extractvalue { ptr, ptr } %3, 1
// CHECK-NEXT:   %5 = extractvalue { ptr, ptr } %3, 0
// CHECK-NEXT:   %6 = call { ptr, ptr } %5(ptr %4, i64 1)
// CHECK-NEXT:   %7 = extractvalue { ptr, ptr } %6, 1
// CHECK-NEXT:   %8 = extractvalue { ptr, ptr } %6, 0
// CHECK-NEXT:   call void %8(ptr %7, i64 2)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define { ptr, ptr } @"{{.*}}/cl/_testgo/closure2.main$1"(ptr %0, i64 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = load { ptr }, ptr %0, align 8
// CHECK-NEXT:   %3 = extractvalue { ptr } %2, 0
// CHECK-NEXT:   %4 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   %5 = getelementptr inbounds { ptr }, ptr %4, i32 0, i32 0
// CHECK-NEXT:   store ptr %3, ptr %5, align 8
// CHECK-NEXT:   %6 = insertvalue { ptr, ptr } { ptr @"{{.*}}/cl/_testgo/closure2.main$1$1", ptr undef }, ptr %4, 1
// CHECK-NEXT:   ret { ptr, ptr } %6
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/closure2.main$1$1"(ptr %0, i64 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = load { ptr }, ptr %0, align 8
// CHECK-NEXT:   %3 = extractvalue { ptr } %2, 0
// CHECK-NEXT:   %4 = load i64, ptr %3, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 7 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %1)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %4)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
