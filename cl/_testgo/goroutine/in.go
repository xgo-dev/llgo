// LITTEST
package main

// CHECK-LINE: @0 = private unnamed_addr constant [5 x i8] c"hello", align 1
// CHECK-LINE: @1 = private unnamed_addr constant [16 x i8] c"Hello, goroutine", align 1
// CHECK-LINE: @2 = private unnamed_addr constant [1 x i8] c".", align 1

func main() {
	done := false
	go println("hello")
	go func(s string) {
		println(s)
		done = true
	}("Hello, goroutine")
	for !done {
		print(".")
	}
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/goroutine.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testgo/goroutine.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testgo/goroutine.init$guard", align 1
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/goroutine.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 1)
// CHECK-NEXT:   store i1 false, ptr %0, align 1
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   %2 = getelementptr inbounds { %"{{.*}}/runtime/internal/runtime.String" }, ptr %1, i32 0, i32 0
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 5 }, ptr %2, align 8
// CHECK-NEXT:   %3 = alloca i8, i64 8, align 1
// CHECK-NEXT:   %4 = call i32 @"{{.*}}/runtime/internal/runtime.CreateThread"(ptr %3, ptr null, ptr @"{{.*}}/cl/_testgo/goroutine._llgo_routine$1", ptr %1)
// CHECK-NEXT:   %5 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   %6 = getelementptr inbounds { ptr }, ptr %5, i32 0, i32 0
// CHECK-NEXT:   store ptr %0, ptr %6, align 8
// CHECK-NEXT:   %7 = insertvalue { ptr, ptr } { ptr @"{{.*}}/cl/_testgo/goroutine.main$1", ptr undef }, ptr %5, 1
// CHECK-NEXT:   %8 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 32)
// CHECK-NEXT:   %9 = getelementptr inbounds { { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" }, ptr %8, i32 0, i32 0
// CHECK-NEXT:   store { ptr, ptr } %7, ptr %9, align 8
// CHECK-NEXT:   %10 = getelementptr inbounds { { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" }, ptr %8, i32 0, i32 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @1, i64 16 }, ptr %10, align 8
// CHECK-NEXT:   %11 = alloca i8, i64 8, align 1
// CHECK-NEXT:   %12 = call i32 @"{{.*}}/runtime/internal/runtime.CreateThread"(ptr %11, ptr null, ptr @"{{.*}}/cl/_testgo/goroutine._llgo_routine$2", ptr %8)
// CHECK-NEXT:   br label %_llgo_3
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_3
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @2, i64 1 })
// CHECK-NEXT:   br label %_llgo_3
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_3
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   %13 = load i1, ptr %0, align 1
// CHECK-NEXT:   br i1 %13, label %_llgo_2, label %_llgo_1
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/goroutine.main$1"(ptr %0, %"{{.*}}/runtime/internal/runtime.String" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %1)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %2 = load { ptr }, ptr %0, align 8
// CHECK-NEXT:   %3 = extractvalue { ptr } %2, 0
// CHECK-NEXT:   store i1 true, ptr %3, align 1
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define ptr @"{{.*}}/cl/_testgo/goroutine._llgo_routine$1"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = load { %"{{.*}}/runtime/internal/runtime.String" }, ptr %0, align 8
// CHECK-NEXT:   %2 = extractvalue { %"{{.*}}/runtime/internal/runtime.String" } %1, 0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %2)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret ptr null
// CHECK-NEXT: }

// CHECK-LABEL: define ptr @"{{.*}}/cl/_testgo/goroutine._llgo_routine$2"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = load { { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" }, ptr %0, align 8
// CHECK-NEXT:   %2 = extractvalue { { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" } %1, 0
// CHECK-NEXT:   %3 = extractvalue { { ptr, ptr }, %"{{.*}}/runtime/internal/runtime.String" } %1, 1
// CHECK-NEXT:   %4 = extractvalue { ptr, ptr } %2, 1
// CHECK-NEXT:   %5 = extractvalue { ptr, ptr } %2, 0
// CHECK-NEXT:   call void %5(ptr %4, %"{{.*}}/runtime/internal/runtime.String" %3)
// CHECK-NEXT:   ret ptr null
// CHECK-NEXT: }
