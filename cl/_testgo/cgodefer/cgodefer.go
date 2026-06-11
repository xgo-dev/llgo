// LITTEST
package main

/*
#include <stdlib.h>
*/
import "C"

func main() {
	p := C.malloc(1024)
	defer C.free(p)
}

// CHECK-LABEL: define [0 x i8] @"{{.*}}/cl/_testgo/cgodefer._Cfunc_free"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %2 = load ptr, ptr @"{{.*}}/cl/_testgo/cgodefer._cgo_{{.*}}_Cfunc_free", align 8
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call [0 x i8] %3(ptr %0)
// CHECK-NEXT:   ret [0 x i8] %4
// CHECK-NEXT: }

// CHECK-LABEL: define ptr @"{{.*}}/cl/_testgo/cgodefer._Cgo_ptr"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   ret ptr %0
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/cgodefer.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testgo/cgodefer.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testgo/cgodefer.init$guard", align 1
// CHECK-NEXT:   call void @syscall.init()
// CHECK-NEXT:   store ptr @_cgo_{{.*}}_Cfunc_free, ptr @"{{.*}}/cl/_testgo/cgodefer._cgo_{{.*}}_Cfunc_free", align 8
// CHECK-NEXT:   store ptr @_cgo_{{.*}}_Cfunc__Cmalloc, ptr @"{{.*}}/cl/_testgo/cgodefer._cgo_{{.*}}_Cfunc__Cmalloc", align 8
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
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
// CHECK-NEXT:   %10 = call ptr @llvm.frameaddress.p0(i32 0)
// CHECK-NEXT:   %11 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 56)
// CHECK-NEXT:   %12 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %11, i32 0, i32 0
// CHECK-NEXT:   store ptr %9, ptr %12, align 8
// CHECK-NEXT:   %13 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %11, i32 0, i32 1
// CHECK-NEXT:   store i64 0, ptr %13, align 8
// CHECK-NEXT:   %14 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %11, i32 0, i32 2
// CHECK-NEXT:   store ptr %8, ptr %14, align 8
// CHECK-NEXT:   %15 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %11, i32 0, i32 3
// CHECK-NEXT:   store ptr blockaddress(@"{{.*}}/cl/_testgo/cgodefer.main", %_llgo_2), ptr %15, align 8
// CHECK-NEXT:   %16 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %11, i32 0, i32 4
// CHECK-NEXT:   store ptr null, ptr %16, align 8
// CHECK-NEXT:   %17 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %11, i32 0, i32 5
// CHECK-NEXT:   store ptr null, ptr %17, align 8
// CHECK-NEXT:   %18 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %11, i32 0, i32 6
// CHECK-NEXT:   store ptr %10, ptr %18, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.SetThreadDefer"(ptr %11)
// CHECK-NEXT:   %19 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %11, i32 0, i32 1
// CHECK-NEXT:   %20 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %11, i32 0, i32 3
// CHECK-NEXT:   %21 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %11, i32 0, i32 4
// CHECK-NEXT:   %22 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %11, i32 0, i32 5
// CHECK-NEXT:   store ptr null, ptr %22, align 8
// CHECK-NEXT:   %23 = call i32 @{{.*}}sigsetjmp(ptr %9, i32 0)
// CHECK-NEXT:   %24 = icmp eq i32 %23, 0
// CHECK-NEXT:   br i1 %24, label %_llgo_4, label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_3
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_5, %_llgo_4
// CHECK-NEXT:   store ptr blockaddress(@"{{.*}}/cl/_testgo/cgodefer.main", %_llgo_3), ptr %20, align 8
// CHECK-NEXT:   %25 = load i64, ptr %19, align 8
// CHECK-NEXT:   %26 = load ptr, ptr %22, align 8
// CHECK-NEXT:   %27 = icmp ne ptr %26, null
// CHECK-NEXT:   br i1 %27, label %_llgo_7, label %_llgo_8
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_5, %_llgo_8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Rethrow"(ptr %8)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %28 = load ptr, ptr %22, align 8
// CHECK-NEXT:   %29 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 32)
// CHECK-NEXT:   %30 = getelementptr inbounds { ptr, i64, { ptr, ptr } }, ptr %29, i32 0, i32 0
// CHECK-NEXT:   store ptr %28, ptr %30, align 8
// CHECK-NEXT:   %31 = getelementptr inbounds { ptr, i64, { ptr, ptr } }, ptr %29, i32 0, i32 1
// CHECK-NEXT:   store i64 0, ptr %31, align 8
// CHECK-NEXT:   %32 = getelementptr inbounds { ptr, i64, { ptr, ptr } }, ptr %29, i32 0, i32 2
// CHECK-NEXT:   store { ptr, ptr } %7, ptr %32, align 8
// CHECK-NEXT:   store ptr %29, ptr %22, align 8
// CHECK-NEXT:   store ptr blockaddress(@"{{.*}}/cl/_testgo/cgodefer.main", %_llgo_6), ptr %21, align 8
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store ptr blockaddress(@"{{.*}}/cl/_testgo/cgodefer.main", %_llgo_3), ptr %21, align 8
// CHECK-NEXT:   %33 = load ptr, ptr %20, align 8
// CHECK-NEXT:   indirectbr ptr %33, [label %_llgo_3, label %_llgo_2]
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_8
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_7:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %34 = load ptr, ptr %22, align 8
// CHECK-NEXT:   %35 = load { ptr, i64, { ptr, ptr } }, ptr %34, align 8
// CHECK-NEXT:   %36 = extractvalue { ptr, i64, { ptr, ptr } } %35, 0
// CHECK-NEXT:   store ptr %36, ptr %22, align 8
// CHECK-NEXT:   %37 = extractvalue { ptr, i64, { ptr, ptr } } %35, 2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.FreeDeferNode"(ptr %34)
// CHECK-NEXT:   %38 = call ptr @llvm.frameaddress.p0(i32 0)
// CHECK-NEXT:   %39 = call ptr @"{{.*}}/runtime/internal/runtime.StartRecoverFrame"(ptr %38)
// CHECK-NEXT:   %40 = extractvalue { ptr, ptr } %37, 1
// CHECK-NEXT:   %41 = extractvalue { ptr, ptr } %37, 0
// CHECK-NEXT:   call void %41(ptr %40)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.EndRecoverFrame"(ptr %39)
// CHECK-NEXT:   br label %_llgo_8
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_8:                                          ; preds = %_llgo_7, %_llgo_2
// CHECK-NEXT:   %42 = load %"{{.*}}/runtime/internal/runtime.Defer", ptr %11, align 8
// CHECK-NEXT:   %43 = extractvalue %"{{.*}}/runtime/internal/runtime.Defer" %42, 2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.SetThreadDefer"(ptr %43)
// CHECK-NEXT:   %44 = load ptr, ptr %21, align 8
// CHECK-NEXT:   indirectbr ptr %44, [label %_llgo_3, label %_llgo_6]
// CHECK-NEXT: }

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
