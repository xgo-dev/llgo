// LITTEST
package main

/*
#include "in.h"
*/
import "C"
import "fmt"

// CHECK-LINE: @2 = private unnamed_addr constant [19 x i8] c"test_structs failed", align 1

func main() {
	r := C.test_structs(&C.s4{a: 1}, &C.s8{a: 1, b: 2}, &C.s12{a: 1, b: 2, c: 3}, &C.s16{a: 1, b: 2, c: 3, d: 4}, &C.s20{a: 1, b: 2, c: 3, d: 4, e: 5})
	fmt.Println(r)
	if r != 35 {
		panic("test_structs failed")
	}
}

// CHECK-LABEL: define i32 @"{{.*}}/cl/_testgo/cgocfiles._Cfunc_test_structs"(ptr %0, ptr %1, ptr %2, ptr %3, ptr %4){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %5 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %6 = load ptr, ptr @"{{.*}}/cl/_testgo/cgocfiles._cgo_1ff9f0e4ef40_Cfunc_test_structs", align 8
// CHECK-NEXT:   %7 = load ptr, ptr %6, align 8
// CHECK-NEXT:   %8 = call i32 %7(ptr %0, ptr %1, ptr %2, ptr %3, ptr %4)
// CHECK-NEXT:   ret i32 %8
// CHECK-NEXT: }

// CHECK-LABEL: define ptr @"{{.*}}/cl/_testgo/cgocfiles._Cgo_ptr"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   ret ptr %0
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/cgocfiles.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testgo/cgocfiles.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testgo/cgocfiles.init$guard", align 1
// CHECK-NEXT:   call void @syscall.init()
// CHECK-NEXT:   call void @fmt.init()
// CHECK-NEXT:   store ptr @_cgo_1ff9f0e4ef40_Cfunc_test_structs, ptr @"{{.*}}/cl/_testgo/cgocfiles._cgo_1ff9f0e4ef40_Cfunc_test_structs", align 8
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/cgocfiles.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 4)
// CHECK-NEXT:   %1 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %1)
// CHECK-NEXT:   %2 = getelementptr inbounds %"{{.*}}/cl/_testgo/cgocfiles._Ctype_struct___3", ptr %0, i32 0, i32 0
// CHECK-NEXT:   store i32 1, ptr %2, align 4
// CHECK-NEXT:   %3 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %4 = icmp eq ptr %3, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %4)
// CHECK-NEXT:   %5 = getelementptr inbounds %"{{.*}}/cl/_testgo/cgocfiles._Ctype_struct___4", ptr %3, i32 0, i32 0
// CHECK-NEXT:   %6 = icmp eq ptr %3, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %6)
// CHECK-NEXT:   %7 = getelementptr inbounds %"{{.*}}/cl/_testgo/cgocfiles._Ctype_struct___4", ptr %3, i32 0, i32 1
// CHECK-NEXT:   store i32 1, ptr %5, align 4
// CHECK-NEXT:   store i32 2, ptr %7, align 4
// CHECK-NEXT:   %8 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 12)
// CHECK-NEXT:   %9 = icmp eq ptr %8, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %9)
// CHECK-NEXT:   %10 = getelementptr inbounds %"{{.*}}/cl/_testgo/cgocfiles._Ctype_struct___0", ptr %8, i32 0, i32 0
// CHECK-NEXT:   %11 = icmp eq ptr %8, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %11)
// CHECK-NEXT:   %12 = getelementptr inbounds %"{{.*}}/cl/_testgo/cgocfiles._Ctype_struct___0", ptr %8, i32 0, i32 1
// CHECK-NEXT:   %13 = icmp eq ptr %8, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %13)
// CHECK-NEXT:   %14 = getelementptr inbounds %"{{.*}}/cl/_testgo/cgocfiles._Ctype_struct___0", ptr %8, i32 0, i32 2
// CHECK-NEXT:   store i32 1, ptr %10, align 4
// CHECK-NEXT:   store i32 2, ptr %12, align 4
// CHECK-NEXT:   store i32 3, ptr %14, align 4
// CHECK-NEXT:   %15 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %16 = icmp eq ptr %15, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %16)
// CHECK-NEXT:   %17 = getelementptr inbounds %"{{.*}}/cl/_testgo/cgocfiles._Ctype_struct___1", ptr %15, i32 0, i32 0
// CHECK-NEXT:   %18 = icmp eq ptr %15, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %18)
// CHECK-NEXT:   %19 = getelementptr inbounds %"{{.*}}/cl/_testgo/cgocfiles._Ctype_struct___1", ptr %15, i32 0, i32 1
// CHECK-NEXT:   %20 = icmp eq ptr %15, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %20)
// CHECK-NEXT:   %21 = getelementptr inbounds %"{{.*}}/cl/_testgo/cgocfiles._Ctype_struct___1", ptr %15, i32 0, i32 2
// CHECK-NEXT:   %22 = icmp eq ptr %15, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %22)
// CHECK-NEXT:   %23 = getelementptr inbounds %"{{.*}}/cl/_testgo/cgocfiles._Ctype_struct___1", ptr %15, i32 0, i32 3
// CHECK-NEXT:   store i32 1, ptr %17, align 4
// CHECK-NEXT:   store i32 2, ptr %19, align 4
// CHECK-NEXT:   store i32 3, ptr %21, align 4
// CHECK-NEXT:   store i32 4, ptr %23, align 4
// CHECK-NEXT:   %24 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 20)
// CHECK-NEXT:   %25 = icmp eq ptr %24, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %25)
// CHECK-NEXT:   %26 = getelementptr inbounds %"{{.*}}/cl/_testgo/cgocfiles._Ctype_struct___2", ptr %24, i32 0, i32 0
// CHECK-NEXT:   %27 = icmp eq ptr %24, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %27)
// CHECK-NEXT:   %28 = getelementptr inbounds %"{{.*}}/cl/_testgo/cgocfiles._Ctype_struct___2", ptr %24, i32 0, i32 1
// CHECK-NEXT:   %29 = icmp eq ptr %24, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %29)
// CHECK-NEXT:   %30 = getelementptr inbounds %"{{.*}}/cl/_testgo/cgocfiles._Ctype_struct___2", ptr %24, i32 0, i32 2
// CHECK-NEXT:   %31 = icmp eq ptr %24, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %31)
// CHECK-NEXT:   %32 = getelementptr inbounds %"{{.*}}/cl/_testgo/cgocfiles._Ctype_struct___2", ptr %24, i32 0, i32 3
// CHECK-NEXT:   %33 = icmp eq ptr %24, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %33)
// CHECK-NEXT:   %34 = getelementptr inbounds %"{{.*}}/cl/_testgo/cgocfiles._Ctype_struct___2", ptr %24, i32 0, i32 4
// CHECK-NEXT:   store i32 1, ptr %26, align 4
// CHECK-NEXT:   store i32 2, ptr %28, align 4
// CHECK-NEXT:   store i32 3, ptr %30, align 4
// CHECK-NEXT:   store i32 4, ptr %32, align 4
// CHECK-NEXT:   store i32 5, ptr %34, align 4
// CHECK-NEXT:   %35 = call i32 @"{{.*}}/cl/_testgo/cgocfiles._Cfunc_test_structs"(ptr %0, ptr %3, ptr %8, ptr %15, ptr %24)
// CHECK-NEXT:   %36 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %37 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %36, i64 0
// CHECK-NEXT:   %38 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store i32 %35, ptr %38, align 4
// CHECK-NEXT:   %39 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"_llgo_{{.*}}/cl/_testgo/cgocfiles._Ctype_int", ptr undef }, ptr %38, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %39, ptr %37, align 8
// CHECK-NEXT:   %40 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %36, 0
// CHECK-NEXT:   %41 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %40, i64 1, 1
// CHECK-NEXT:   %42 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %41, i64 1, 2
// CHECK-NEXT:   %43 = call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @fmt.Println(%"{{.*}}/runtime/internal/runtime.Slice" %42)
// CHECK-NEXT:   %44 = icmp ne i32 %35, 35
// CHECK-NEXT:   br i1 %44, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %45 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @2, i64 19 }, ptr %45, align 8
// CHECK-NEXT:   %46 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %45, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %46)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
