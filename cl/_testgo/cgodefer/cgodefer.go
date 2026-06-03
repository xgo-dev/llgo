// LITTEST
package main

/*
#include <stdlib.h>
*/
import "C"

// CHECK-LABEL: define [0 x i8] @"{{.*}}/cl/_testgo/cgodefer._Cfunc_free"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %2 = load ptr, ptr @"{{.*}}/cl/_testgo/cgodefer._cgo_{{.*}}_Cfunc_free", align 8
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call [0 x i8] %3(ptr %0)
// CHECK-NEXT:   ret [0 x i8] %4
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/cgodefer.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %1 = call ptr @malloc(i64 1024)
// CHECK-NEXT:   store ptr %1, ptr %0, align 8
// CHECK-NEXT:   %2 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   %3 = getelementptr inbounds { ptr }, ptr %2, i32 0, i32 0
// CHECK-NEXT:   store ptr %0, ptr %3, align 8
// CHECK-NEXT:   %4 = insertvalue { ptr, ptr } { ptr @"{{.*}}/cl/_testgo/cgodefer.main$1", ptr undef }, ptr %2, 1
// CHECK-NEXT:   %5 = extractvalue { ptr, ptr } %4, 1
// CHECK-NEXT:   %6 = extractvalue { ptr, ptr } %4, 0
// CHECK-NEXT:   %7 = call { ptr, ptr } %6(ptr %5)
// CHECK-NEXT:   %8 = call ptr @"{{.*}}/runtime/internal/runtime.GetThreadDefer"()
// CHECK-NEXT:   %9 = alloca i8, i64 {{.*}}, align 1
// CHECK-NEXT:   %10 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 48)
// CHECK-NEXT:   %11 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %10, i32 0, i32 0
// CHECK-NEXT:   store ptr %9, ptr %11, align 8
// CHECK-NEXT:   %12 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %10, i32 0, i32 1
// CHECK-NEXT:   store i64 0, ptr %12, align 8
// CHECK-NEXT:   %13 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %10, i32 0, i32 2
// CHECK-NEXT:   store ptr %8, ptr %13, align 8
// CHECK-NEXT:   %14 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %10, i32 0, i32 3
// CHECK-NEXT:   store ptr blockaddress(@"{{.*}}/cl/_testgo/cgodefer.main", %_llgo_2), ptr %14, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.SetThreadDefer"(ptr %10)
// CHECK-NEXT:   %15 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %10, i32 0, i32 1
// CHECK-NEXT:   %16 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %10, i32 0, i32 3
// CHECK-NEXT:   %17 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %10, i32 0, i32 4
// CHECK-NEXT:   %18 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %10, i32 0, i32 5
// CHECK-NEXT:   store ptr null, ptr %18, align 8
// CHECK-NEXT:   %19 = call i32 @{{.*}}sigsetjmp(ptr %9, i32 0)
// CHECK-NEXT:   %20 = icmp eq i32 %19, 0
// CHECK-NEXT:   br i1 %20, label %_llgo_4, label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_3
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_5, %_llgo_4
// CHECK-NEXT:   store ptr blockaddress(@"{{.*}}/cl/_testgo/cgodefer.main", %_llgo_3), ptr %16, align 8
// CHECK-NEXT:   %21 = load i64, ptr %15, align 8
// CHECK-NEXT:   %22 = load ptr, ptr %18, align 8
// CHECK-NEXT:   %23 = icmp ne ptr %22, null
// CHECK-NEXT:   br i1 %23, label %_llgo_7, label %_llgo_8
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_5, %_llgo_8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Rethrow"(ptr %8)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %24 = load ptr, ptr %18, align 8
// CHECK-NEXT:   %25 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 32)
// CHECK-NEXT:   %26 = getelementptr inbounds { ptr, i64, { ptr, ptr } }, ptr %25, i32 0, i32 0
// CHECK-NEXT:   store ptr %24, ptr %26, align 8
// CHECK-NEXT:   %27 = getelementptr inbounds { ptr, i64, { ptr, ptr } }, ptr %25, i32 0, i32 1
// CHECK-NEXT:   store i64 0, ptr %27, align 8
// CHECK-NEXT:   %28 = getelementptr inbounds { ptr, i64, { ptr, ptr } }, ptr %25, i32 0, i32 2
// CHECK-NEXT:   store { ptr, ptr } %7, ptr %28, align 8
// CHECK-NEXT:   store ptr %25, ptr %18, align 8
// CHECK-NEXT:   store ptr blockaddress(@"{{.*}}/cl/_testgo/cgodefer.main", %_llgo_6), ptr %17, align 8
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store ptr blockaddress(@"{{.*}}/cl/_testgo/cgodefer.main", %_llgo_3), ptr %17, align 8
// CHECK-NEXT:   %29 = load ptr, ptr %16, align 8
// CHECK-NEXT:   indirectbr ptr %29, [label %_llgo_3, label %_llgo_2]
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_8
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_7:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %30 = load ptr, ptr %18, align 8
// CHECK-NEXT:   %31 = load { ptr, i64, { ptr, ptr } }, ptr %30, align 8
// CHECK-NEXT:   %32 = extractvalue { ptr, i64, { ptr, ptr } } %31, 0
// CHECK-NEXT:   store ptr %32, ptr %18, align 8
// CHECK-NEXT:   %33 = extractvalue { ptr, i64, { ptr, ptr } } %31, 2
// CHECK-NEXT:   %34 = extractvalue { ptr, ptr } %33, 1
// CHECK-NEXT:   %35 = extractvalue { ptr, ptr } %33, 0
// CHECK-NEXT:   call void %35(ptr %34)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.FreeDeferNode"(ptr %30)
// CHECK-NEXT:   br label %_llgo_8
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_8:                                          ; preds = %_llgo_7, %_llgo_2
// CHECK-NEXT:   %36 = load %"{{.*}}/runtime/internal/runtime.Defer", ptr %10, align 8
// CHECK-NEXT:   %37 = extractvalue %"{{.*}}/runtime/internal/runtime.Defer" %36, 2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.SetThreadDefer"(ptr %37)
// CHECK-NEXT:   %38 = load ptr, ptr %17, align 8
// CHECK-NEXT:   indirectbr ptr %38, [label %_llgo_3, label %_llgo_6]
// CHECK-NEXT: }
func main() {
	// CHECK-LABEL: define { ptr, ptr } @"{{.*}}/cl/_testgo/cgodefer.main$1"(ptr %0){{.*}} {
	// CHECK-NEXT: _llgo_0:
	// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
	// CHECK-NEXT:   %2 = load { ptr }, ptr %0, align 8
	// CHECK-NEXT:   %3 = extractvalue { ptr } %2, 0
	// CHECK-NEXT:   %4 = load ptr, ptr %3, align 8
	// CHECK-NEXT:   store ptr %4, ptr %1, align 8
	// CHECK-NEXT:   %5 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
	// CHECK-NEXT:   %6 = getelementptr inbounds { ptr }, ptr %5, i32 0, i32 0
	// CHECK-NEXT:   store ptr %1, ptr %6, align 8
	// CHECK-NEXT:   %7 = insertvalue { ptr, ptr } { ptr @"{{.*}}/cl/_testgo/cgodefer.main$1$1", ptr undef }, ptr %5, 1
	// CHECK-NEXT:   ret { ptr, ptr } %7
	// CHECK-NEXT: }
	p := C.malloc(1024)
	// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/cgodefer.main$1$1"(ptr %0){{.*}} {
	// CHECK-NEXT: _llgo_0:
	// CHECK-NEXT:   %1 = load { ptr }, ptr %0, align 8
	// CHECK-NEXT:   %2 = extractvalue { ptr } %1, 0
	// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
	// CHECK-NEXT:   %4 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_Pointer, ptr undef }, ptr %3, 1
	// CHECK-NEXT:   %5 = extractvalue { ptr } %1, 0
	// CHECK-NEXT:   %6 = load ptr, ptr %5, align 8
	// CHECK-NEXT:   %7 = call [0 x i8] @"{{.*}}/cl/_testgo/cgodefer._Cfunc_free"(ptr %6)
	// CHECK-NEXT:   ret void
	// CHECK-NEXT: }
	defer C.free(p)
}
