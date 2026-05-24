// LITTEST
package main

import (
	"unsafe"

	"github.com/goplus/llgo/runtime/abi"
)

// CHECK-LINE: @121 = private unnamed_addr constant [13 x i8] c"error field 0", align 1
// CHECK-LINE: @122 = private unnamed_addr constant [18 x i8] c"error field 0 elem", align 1
// CHECK-LINE: @123 = private unnamed_addr constant [13 x i8] c"error field 1", align 1
// CHECK-LINE: @124 = private unnamed_addr constant [18 x i8] c"error field 1 elem", align 1
// CHECK-LINE: @125 = private unnamed_addr constant [13 x i8] c"error field 2", align 1
// CHECK-LINE: @126 = private unnamed_addr constant [13 x i8] c"error field 3", align 1

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/abinamed.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testrt/abinamed.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testrt/abinamed.init$guard", align 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/abi.init"()
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

type T struct {
	p *T
	t *abi.Type
	n uintptr
	a []T
}

type eface struct {
	typ  *abi.Type
	data unsafe.Pointer
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/abinamed.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 48)
// CHECK-NEXT:   store %"{{.*}}/cl/_testrt/abinamed.T" zeroinitializer, ptr %0, align 8
// CHECK-NEXT:   %1 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"_llgo_{{.*}}/cl/_testrt/abinamed.T", ptr undef }, ptr %0, 1
// CHECK-NEXT:   %2 = call ptr @"{{.*}}/cl/_testrt/abinamed.toEface"(%"{{.*}}/runtime/internal/runtime.eface" %1)
// CHECK-NEXT:   %3 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 72)
// CHECK-NEXT:   store %"{{.*}}/runtime/abi.Type" zeroinitializer, ptr %3, align 8
// CHECK-NEXT:   %4 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"_llgo_{{.*}}/runtime/abi.Type", ptr undef }, ptr %3, 1
// CHECK-NEXT:   %5 = call ptr @"{{.*}}/cl/_testrt/abinamed.toEface"(%"{{.*}}/runtime/internal/runtime.eface" %4)
// CHECK-NEXT:   %6 = icmp eq ptr %2, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %6)
// CHECK-NEXT:   %7 = getelementptr inbounds %"{{.*}}/cl/_testrt/abinamed.eface", ptr %2, i32 0, i32 0
// CHECK-NEXT:   %8 = load ptr, ptr %7, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintPointer"(ptr %8)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %9 = icmp eq ptr %2, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %9)
// CHECK-NEXT:   %10 = getelementptr inbounds %"{{.*}}/cl/_testrt/abinamed.eface", ptr %2, i32 0, i32 0
// CHECK-NEXT:   %11 = load ptr, ptr %10, align 8
// CHECK-NEXT:   %12 = icmp eq ptr %11, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %12)
// CHECK-NEXT:   %13 = getelementptr inbounds %"{{.*}}/runtime/abi.Type", ptr %11, i32 0, i32 10
// CHECK-NEXT:   %14 = load ptr, ptr %13, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintPointer"(ptr %14)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %15 = icmp eq ptr %5, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %15)
// CHECK-NEXT:   %16 = getelementptr inbounds %"{{.*}}/cl/_testrt/abinamed.eface", ptr %5, i32 0, i32 0
// CHECK-NEXT:   %17 = load ptr, ptr %16, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintPointer"(ptr %17)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %18 = icmp eq ptr %5, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %18)
// CHECK-NEXT:   %19 = getelementptr inbounds %"{{.*}}/cl/_testrt/abinamed.eface", ptr %5, i32 0, i32 0
// CHECK-NEXT:   %20 = load ptr, ptr %19, align 8
// CHECK-NEXT:   %21 = icmp eq ptr %20, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %21)
// CHECK-NEXT:   %22 = getelementptr inbounds %"{{.*}}/runtime/abi.Type", ptr %20, i32 0, i32 10
// CHECK-NEXT:   %23 = load ptr, ptr %22, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintPointer"(ptr %23)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %24 = alloca %"{{.*}}/runtime/abi.StructField", align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %24, i8 0, i64 56, i1 false)
// CHECK-NEXT:   %25 = icmp eq ptr %2, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %25)
// CHECK-NEXT:   %26 = getelementptr inbounds %"{{.*}}/cl/_testrt/abinamed.eface", ptr %2, i32 0, i32 0
// CHECK-NEXT:   %27 = load ptr, ptr %26, align 8
// CHECK-NEXT:   %28 = call ptr @"{{.*}}/runtime/abi.(*Type).StructType"(ptr %27)
// CHECK-NEXT:   %29 = icmp eq ptr %28, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %29)
// CHECK-NEXT:   %30 = getelementptr inbounds %"{{.*}}/runtime/abi.StructType", ptr %28, i32 0, i32 2
// CHECK-NEXT:   %31 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %30, align 8
// CHECK-NEXT:   %32 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %31, 0
// CHECK-NEXT:   %33 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %31, 1
// CHECK-NEXT:   %34 = icmp uge i64 0, %33
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.CheckIndexRange"(i1 %34, i64 0, i1 true, i64 %33)
// CHECK-NEXT:   %35 = getelementptr inbounds %"{{.*}}/runtime/abi.StructField", ptr %32, i64 0
// CHECK-NEXT:   %36 = load %"{{.*}}/runtime/abi.StructField", ptr %35, align 8
// CHECK-NEXT:   store %"{{.*}}/runtime/abi.StructField" %36, ptr %24, align 8
// CHECK-NEXT:   %37 = icmp eq ptr %24, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %37)
// CHECK-NEXT:   %38 = getelementptr inbounds %"{{.*}}/runtime/abi.StructField", ptr %24, i32 0, i32 1
// CHECK-NEXT:   %39 = load ptr, ptr %38, align 8
// CHECK-NEXT:   %40 = icmp eq ptr %2, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %40)
// CHECK-NEXT:   %41 = getelementptr inbounds %"{{.*}}/cl/_testrt/abinamed.eface", ptr %2, i32 0, i32 0
// CHECK-NEXT:   %42 = load ptr, ptr %41, align 8
// CHECK-NEXT:   %43 = icmp eq ptr %42, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %43)
// CHECK-NEXT:   %44 = getelementptr inbounds %"{{.*}}/runtime/abi.Type", ptr %42, i32 0, i32 10
// CHECK-NEXT:   %45 = load ptr, ptr %44, align 8
// CHECK-NEXT:   %46 = icmp ne ptr %39, %45
// CHECK-NEXT:   br i1 %46, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %47 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @121, i64 13 }, ptr %47, align 8
// CHECK-NEXT:   %48 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %47, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %48)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %49 = icmp eq ptr %24, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %49)
// CHECK-NEXT:   %50 = getelementptr inbounds %"{{.*}}/runtime/abi.StructField", ptr %24, i32 0, i32 1
// CHECK-NEXT:   %51 = load ptr, ptr %50, align 8
// CHECK-NEXT:   %52 = call ptr @"{{.*}}/runtime/abi.(*Type).Elem"(ptr %51)
// CHECK-NEXT:   %53 = icmp eq ptr %2, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %53)
// CHECK-NEXT:   %54 = getelementptr inbounds %"{{.*}}/cl/_testrt/abinamed.eface", ptr %2, i32 0, i32 0
// CHECK-NEXT:   %55 = load ptr, ptr %54, align 8
// CHECK-NEXT:   %56 = icmp ne ptr %52, %55
// CHECK-NEXT:   br i1 %56, label %_llgo_3, label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %57 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @122, i64 18 }, ptr %57, align 8
// CHECK-NEXT:   %58 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %57, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %58)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %59 = alloca %"{{.*}}/runtime/abi.StructField", align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %59, i8 0, i64 56, i1 false)
// CHECK-NEXT:   %60 = icmp eq ptr %2, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %60)
// CHECK-NEXT:   %61 = getelementptr inbounds %"{{.*}}/cl/_testrt/abinamed.eface", ptr %2, i32 0, i32 0
// CHECK-NEXT:   %62 = load ptr, ptr %61, align 8
// CHECK-NEXT:   %63 = call ptr @"{{.*}}/runtime/abi.(*Type).StructType"(ptr %62)
// CHECK-NEXT:   %64 = icmp eq ptr %63, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %64)
// CHECK-NEXT:   %65 = getelementptr inbounds %"{{.*}}/runtime/abi.StructType", ptr %63, i32 0, i32 2
// CHECK-NEXT:   %66 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %65, align 8
// CHECK-NEXT:   %67 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %66, 0
// CHECK-NEXT:   %68 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %66, 1
// CHECK-NEXT:   %69 = icmp uge i64 1, %68
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.CheckIndexRange"(i1 %69, i64 1, i1 true, i64 %68)
// CHECK-NEXT:   %70 = getelementptr inbounds %"{{.*}}/runtime/abi.StructField", ptr %67, i64 1
// CHECK-NEXT:   %71 = load %"{{.*}}/runtime/abi.StructField", ptr %70, align 8
// CHECK-NEXT:   store %"{{.*}}/runtime/abi.StructField" %71, ptr %59, align 8
// CHECK-NEXT:   %72 = icmp eq ptr %59, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %72)
// CHECK-NEXT:   %73 = getelementptr inbounds %"{{.*}}/runtime/abi.StructField", ptr %59, i32 0, i32 1
// CHECK-NEXT:   %74 = load ptr, ptr %73, align 8
// CHECK-NEXT:   %75 = icmp eq ptr %5, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %75)
// CHECK-NEXT:   %76 = getelementptr inbounds %"{{.*}}/cl/_testrt/abinamed.eface", ptr %5, i32 0, i32 0
// CHECK-NEXT:   %77 = load ptr, ptr %76, align 8
// CHECK-NEXT:   %78 = icmp eq ptr %77, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %78)
// CHECK-NEXT:   %79 = getelementptr inbounds %"{{.*}}/runtime/abi.Type", ptr %77, i32 0, i32 10
// CHECK-NEXT:   %80 = load ptr, ptr %79, align 8
// CHECK-NEXT:   %81 = icmp ne ptr %74, %80
// CHECK-NEXT:   br i1 %81, label %_llgo_5, label %_llgo_6
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_4
// CHECK-NEXT:   %82 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @123, i64 13 }, ptr %82, align 8
// CHECK-NEXT:   %83 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %82, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %83)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_4
// CHECK-NEXT:   %84 = icmp eq ptr %59, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %84)
// CHECK-NEXT:   %85 = getelementptr inbounds %"{{.*}}/runtime/abi.StructField", ptr %59, i32 0, i32 1
// CHECK-NEXT:   %86 = load ptr, ptr %85, align 8
// CHECK-NEXT:   %87 = call ptr @"{{.*}}/runtime/abi.(*Type).Elem"(ptr %86)
// CHECK-NEXT:   %88 = icmp eq ptr %5, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %88)
// CHECK-NEXT:   %89 = getelementptr inbounds %"{{.*}}/cl/_testrt/abinamed.eface", ptr %5, i32 0, i32 0
// CHECK-NEXT:   %90 = load ptr, ptr %89, align 8
// CHECK-NEXT:   %91 = icmp ne ptr %87, %90
// CHECK-NEXT:   br i1 %91, label %_llgo_7, label %_llgo_8
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_7:                                          ; preds = %_llgo_6
// CHECK-NEXT:   %92 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @124, i64 18 }, ptr %92, align 8
// CHECK-NEXT:   %93 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %92, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %93)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_8:                                          ; preds = %_llgo_6
// CHECK-NEXT:   %94 = alloca %"{{.*}}/runtime/abi.StructField", align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %94, i8 0, i64 56, i1 false)
// CHECK-NEXT:   %95 = icmp eq ptr %2, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %95)
// CHECK-NEXT:   %96 = getelementptr inbounds %"{{.*}}/cl/_testrt/abinamed.eface", ptr %2, i32 0, i32 0
// CHECK-NEXT:   %97 = load ptr, ptr %96, align 8
// CHECK-NEXT:   %98 = call ptr @"{{.*}}/runtime/abi.(*Type).StructType"(ptr %97)
// CHECK-NEXT:   %99 = icmp eq ptr %98, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %99)
// CHECK-NEXT:   %100 = getelementptr inbounds %"{{.*}}/runtime/abi.StructType", ptr %98, i32 0, i32 2
// CHECK-NEXT:   %101 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %100, align 8
// CHECK-NEXT:   %102 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %101, 0
// CHECK-NEXT:   %103 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %101, 1
// CHECK-NEXT:   %104 = icmp uge i64 2, %103
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.CheckIndexRange"(i1 %104, i64 2, i1 true, i64 %103)
// CHECK-NEXT:   %105 = getelementptr inbounds %"{{.*}}/runtime/abi.StructField", ptr %102, i64 2
// CHECK-NEXT:   %106 = load %"{{.*}}/runtime/abi.StructField", ptr %105, align 8
// CHECK-NEXT:   store %"{{.*}}/runtime/abi.StructField" %106, ptr %94, align 8
// CHECK-NEXT:   %107 = icmp eq ptr %94, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %107)
// CHECK-NEXT:   %108 = getelementptr inbounds %"{{.*}}/runtime/abi.StructField", ptr %94, i32 0, i32 1
// CHECK-NEXT:   %109 = load ptr, ptr %108, align 8
// CHECK-NEXT:   %110 = icmp eq ptr %5, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %110)
// CHECK-NEXT:   %111 = getelementptr inbounds %"{{.*}}/cl/_testrt/abinamed.eface", ptr %5, i32 0, i32 0
// CHECK-NEXT:   %112 = load ptr, ptr %111, align 8
// CHECK-NEXT:   %113 = call ptr @"{{.*}}/runtime/abi.(*Type).StructType"(ptr %112)
// CHECK-NEXT:   %114 = icmp eq ptr %113, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %114)
// CHECK-NEXT:   %115 = getelementptr inbounds %"{{.*}}/runtime/abi.StructType", ptr %113, i32 0, i32 2
// CHECK-NEXT:   %116 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %115, align 8
// CHECK-NEXT:   %117 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %116, 0
// CHECK-NEXT:   %118 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %116, 1
// CHECK-NEXT:   %119 = icmp uge i64 0, %118
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.CheckIndexRange"(i1 %119, i64 0, i1 true, i64 %118)
// CHECK-NEXT:   %120 = getelementptr inbounds %"{{.*}}/runtime/abi.StructField", ptr %117, i64 0
// CHECK-NEXT:   %121 = icmp eq ptr %120, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %121)
// CHECK-NEXT:   %122 = getelementptr inbounds %"{{.*}}/runtime/abi.StructField", ptr %120, i32 0, i32 1
// CHECK-NEXT:   %123 = load ptr, ptr %122, align 8
// CHECK-NEXT:   %124 = icmp ne ptr %109, %123
// CHECK-NEXT:   br i1 %124, label %_llgo_9, label %_llgo_10
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_9:                                          ; preds = %_llgo_8
// CHECK-NEXT:   %125 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @125, i64 13 }, ptr %125, align 8
// CHECK-NEXT:   %126 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %125, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %126)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_10:                                         ; preds = %_llgo_8
// CHECK-NEXT:   %127 = alloca %"{{.*}}/runtime/abi.StructField", align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %127, i8 0, i64 56, i1 false)
// CHECK-NEXT:   %128 = icmp eq ptr %2, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %128)
// CHECK-NEXT:   %129 = getelementptr inbounds %"{{.*}}/cl/_testrt/abinamed.eface", ptr %2, i32 0, i32 0
// CHECK-NEXT:   %130 = load ptr, ptr %129, align 8
// CHECK-NEXT:   %131 = call ptr @"{{.*}}/runtime/abi.(*Type).StructType"(ptr %130)
// CHECK-NEXT:   %132 = icmp eq ptr %131, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %132)
// CHECK-NEXT:   %133 = getelementptr inbounds %"{{.*}}/runtime/abi.StructType", ptr %131, i32 0, i32 2
// CHECK-NEXT:   %134 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %133, align 8
// CHECK-NEXT:   %135 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %134, 0
// CHECK-NEXT:   %136 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %134, 1
// CHECK-NEXT:   %137 = icmp uge i64 3, %136
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.CheckIndexRange"(i1 %137, i64 3, i1 true, i64 %136)
// CHECK-NEXT:   %138 = getelementptr inbounds %"{{.*}}/runtime/abi.StructField", ptr %135, i64 3
// CHECK-NEXT:   %139 = load %"{{.*}}/runtime/abi.StructField", ptr %138, align 8
// CHECK-NEXT:   store %"{{.*}}/runtime/abi.StructField" %139, ptr %127, align 8
// CHECK-NEXT:   %140 = icmp eq ptr %127, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %140)
// CHECK-NEXT:   %141 = getelementptr inbounds %"{{.*}}/runtime/abi.StructField", ptr %127, i32 0, i32 1
// CHECK-NEXT:   %142 = load ptr, ptr %141, align 8
// CHECK-NEXT:   %143 = call ptr @"{{.*}}/runtime/abi.(*Type).Elem"(ptr %142)
// CHECK-NEXT:   %144 = icmp eq ptr %2, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %144)
// CHECK-NEXT:   %145 = getelementptr inbounds %"{{.*}}/cl/_testrt/abinamed.eface", ptr %2, i32 0, i32 0
// CHECK-NEXT:   %146 = load ptr, ptr %145, align 8
// CHECK-NEXT:   %147 = icmp ne ptr %143, %146
// CHECK-NEXT:   br i1 %147, label %_llgo_11, label %_llgo_12
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_11:                                         ; preds = %_llgo_10
// CHECK-NEXT:   %148 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @126, i64 13 }, ptr %148, align 8
// CHECK-NEXT:   %149 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %148, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %149)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_12:                                         ; preds = %_llgo_10
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

func main() {
	e := toEface(T{})
	e2 := toEface(abi.Type{})

	println(e.typ)
	println(e.typ.PtrToThis_)
	println(e2.typ)
	println(e2.typ.PtrToThis_)

	f0 := e.typ.StructType().Fields[0]
	if f0.Typ != e.typ.PtrToThis_ {
		panic("error field 0")
	}
	if f0.Typ.Elem() != e.typ {
		panic("error field 0 elem")
	}
	f1 := e.typ.StructType().Fields[1]
	if f1.Typ != e2.typ.PtrToThis_ {
		panic("error field 1")
	}
	if f1.Typ.Elem() != e2.typ {
		panic("error field 1 elem")
	}
	f2 := e.typ.StructType().Fields[2]
	if f2.Typ != e2.typ.StructType().Fields[0].Typ {
		panic("error field 2")
	}
	f3 := e.typ.StructType().Fields[3]
	if f3.Typ.Elem() != e.typ {
		panic("error field 3")
	}
}

// CHECK-LABEL: define ptr @"{{.*}}/cl/_testrt/abinamed.toEface"(%"{{.*}}/runtime/internal/runtime.eface" %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %0, ptr %1, align 8
// CHECK-NEXT:   ret ptr %1
// CHECK-NEXT: }

func toEface(i any) *eface {
	return (*eface)(unsafe.Pointer(&i))
}

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequal64"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.memequal64"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequal8"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.memequal8"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/runtime/abi.(*Type).Align"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/runtime/abi.(*Type).Align"(ptr %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce ptr @"__llgo_stub.{{.*}}/runtime/abi.(*Type).ArrayType"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call ptr @"{{.*}}/runtime/abi.(*Type).ArrayType"(ptr %1)
// CHECK-NEXT:   ret ptr %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/runtime/abi.(*ArrayType).Align"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/runtime/abi.(*ArrayType).Align"(ptr %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce ptr @"__llgo_stub.{{.*}}/runtime/abi.(*ArrayType).ArrayType"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call ptr @"{{.*}}/runtime/abi.(*ArrayType).ArrayType"(ptr %1)
// CHECK-NEXT:   ret ptr %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/runtime/abi.(*ArrayType).ChanDir"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/runtime/abi.(*ArrayType).ChanDir"(ptr %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce ptr @"__llgo_stub.{{.*}}/runtime/abi.(*ArrayType).Common"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call ptr @"{{.*}}/runtime/abi.(*ArrayType).Common"(ptr %1)
// CHECK-NEXT:   ret ptr %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.Slice" @"__llgo_stub.{{.*}}/runtime/abi.(*ArrayType).ExportedMethods"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/abi.(*ArrayType).ExportedMethods"(ptr %1)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.Slice" %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/abi.(*Method).Exported"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i1 @"{{.*}}/runtime/abi.(*Method).Exported"(ptr %1)
// CHECK-NEXT:   ret i1 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.String" @"__llgo_stub.{{.*}}/runtime/abi.(*Method).Name"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/runtime/abi.(*Method).Name"(ptr %1)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.String" %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.String" @"__llgo_stub.{{.*}}/runtime/abi.(*Method).PkgPath"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/runtime/abi.(*Method).PkgPath"(ptr %1)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.String" %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/runtime/abi.(*FuncType).Align"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/runtime/abi.(*FuncType).Align"(ptr %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce ptr @"__llgo_stub.{{.*}}/runtime/abi.(*FuncType).ArrayType"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call ptr @"{{.*}}/runtime/abi.(*FuncType).ArrayType"(ptr %1)
// CHECK-NEXT:   ret ptr %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/runtime/abi.(*FuncType).ChanDir"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/runtime/abi.(*FuncType).ChanDir"(ptr %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce ptr @"__llgo_stub.{{.*}}/runtime/abi.(*FuncType).Common"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call ptr @"{{.*}}/runtime/abi.(*FuncType).Common"(ptr %1)
// CHECK-NEXT:   ret ptr %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce ptr @"__llgo_stub.{{.*}}/runtime/abi.(*FuncType).Elem"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call ptr @"{{.*}}/runtime/abi.(*FuncType).Elem"(ptr %1)
// CHECK-NEXT:   ret ptr %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.Slice" @"__llgo_stub.{{.*}}/runtime/abi.(*FuncType).ExportedMethods"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/abi.(*FuncType).ExportedMethods"(ptr %1)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.Slice" %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/runtime/abi.(*FuncType).FieldAlign"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/runtime/abi.(*FuncType).FieldAlign"(ptr %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce ptr @"__llgo_stub.{{.*}}/runtime/abi.(*FuncType).FuncType"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call ptr @"{{.*}}/runtime/abi.(*FuncType).FuncType"(ptr %1)
// CHECK-NEXT:   ret ptr %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.Slice" @"__llgo_stub.{{.*}}/runtime/abi.(*FuncType).GcSlice"(ptr %0, ptr %1, i64 %2, i64 %3){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %4 = tail call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/abi.(*FuncType).GcSlice"(ptr %1, i64 %2, i64 %3)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.Slice" %4
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/abi.(*FuncType).HasName"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i1 @"{{.*}}/runtime/abi.(*FuncType).HasName"(ptr %1)
// CHECK-NEXT:   ret i1 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/abi.(*FuncType).IfaceIndir"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i1 @"{{.*}}/runtime/abi.(*FuncType).IfaceIndir"(ptr %1)
// CHECK-NEXT:   ret i1 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce ptr @"__llgo_stub.{{.*}}/runtime/abi.(*FuncType).InterfaceType"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call ptr @"{{.*}}/runtime/abi.(*FuncType).InterfaceType"(ptr %1)
// CHECK-NEXT:   ret ptr %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/abi.(*Imethod).Exported"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i1 @"{{.*}}/runtime/abi.(*Imethod).Exported"(ptr %1)
// CHECK-NEXT:   ret i1 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.String" @"__llgo_stub.{{.*}}/runtime/abi.(*Imethod).Name"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/runtime/abi.(*Imethod).Name"(ptr %1)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.String" %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.String" @"__llgo_stub.{{.*}}/runtime/abi.(*Imethod).PkgPath"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/runtime/abi.(*Imethod).PkgPath"(ptr %1)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.String" %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/runtime/abi.(*InterfaceType).Align"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/runtime/abi.(*InterfaceType).Align"(ptr %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce ptr @"__llgo_stub.{{.*}}/runtime/abi.(*InterfaceType).ArrayType"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call ptr @"{{.*}}/runtime/abi.(*InterfaceType).ArrayType"(ptr %1)
// CHECK-NEXT:   ret ptr %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/runtime/abi.(*InterfaceType).ChanDir"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/runtime/abi.(*InterfaceType).ChanDir"(ptr %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce ptr @"__llgo_stub.{{.*}}/runtime/abi.(*InterfaceType).Common"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call ptr @"{{.*}}/runtime/abi.(*InterfaceType).Common"(ptr %1)
// CHECK-NEXT:   ret ptr %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce ptr @"__llgo_stub.{{.*}}/runtime/abi.(*InterfaceType).Elem"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call ptr @"{{.*}}/runtime/abi.(*InterfaceType).Elem"(ptr %1)
// CHECK-NEXT:   ret ptr %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.Slice" @"__llgo_stub.{{.*}}/runtime/abi.(*InterfaceType).ExportedMethods"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/abi.(*InterfaceType).ExportedMethods"(ptr %1)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.Slice" %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/runtime/abi.(*InterfaceType).FieldAlign"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/runtime/abi.(*InterfaceType).FieldAlign"(ptr %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce ptr @"__llgo_stub.{{.*}}/runtime/abi.(*InterfaceType).FuncType"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call ptr @"{{.*}}/runtime/abi.(*InterfaceType).FuncType"(ptr %1)
// CHECK-NEXT:   ret ptr %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.Slice" @"__llgo_stub.{{.*}}/runtime/abi.(*InterfaceType).GcSlice"(ptr %0, ptr %1, i64 %2, i64 %3){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %4 = tail call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/abi.(*InterfaceType).GcSlice"(ptr %1, i64 %2, i64 %3)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.Slice" %4
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/abi.(*InterfaceType).HasName"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i1 @"{{.*}}/runtime/abi.(*InterfaceType).HasName"(ptr %1)
// CHECK-NEXT:   ret i1 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/abi.(*InterfaceType).IfaceIndir"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i1 @"{{.*}}/runtime/abi.(*InterfaceType).IfaceIndir"(ptr %1)
// CHECK-NEXT:   ret i1 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce ptr @"__llgo_stub.{{.*}}/runtime/abi.(*InterfaceType).InterfaceType"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call ptr @"{{.*}}/runtime/abi.(*InterfaceType).InterfaceType"(ptr %1)
// CHECK-NEXT:   ret ptr %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/abi.(*InterfaceType).IsClosure"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i1 @"{{.*}}/runtime/abi.(*InterfaceType).IsClosure"(ptr %1)
// CHECK-NEXT:   ret i1 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/abi.(*InterfaceType).IsDirectIface"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i1 @"{{.*}}/runtime/abi.(*InterfaceType).IsDirectIface"(ptr %1)
// CHECK-NEXT:   ret i1 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce ptr @"__llgo_stub.{{.*}}/runtime/abi.(*InterfaceType).Key"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call ptr @"{{.*}}/runtime/abi.(*InterfaceType).Key"(ptr %1)
// CHECK-NEXT:   ret ptr %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/runtime/abi.(*InterfaceType).Kind"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/runtime/abi.(*InterfaceType).Kind"(ptr %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.String" @"__llgo_stub.{{.*}}/runtime/abi.(*Kind).String"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/runtime/abi.(*Kind).String"(ptr %1)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.String" %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.String" @"__llgo_stub.{{.*}}/runtime/abi.Kind.String"(ptr %0, i64 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/runtime/abi.Kind.String"(i64 %1)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.String" %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/runtime/abi.(*InterfaceType).Len"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/runtime/abi.(*InterfaceType).Len"(ptr %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce ptr @"__llgo_stub.{{.*}}/runtime/abi.(*InterfaceType).MapType"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call ptr @"{{.*}}/runtime/abi.(*InterfaceType).MapType"(ptr %1)
// CHECK-NEXT:   ret ptr %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequal16"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.memequal16"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/runtime/abi.(*MapType).Align"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/runtime/abi.(*MapType).Align"(ptr %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce ptr @"__llgo_stub.{{.*}}/runtime/abi.(*MapType).ArrayType"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call ptr @"{{.*}}/runtime/abi.(*MapType).ArrayType"(ptr %1)
// CHECK-NEXT:   ret ptr %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/runtime/abi.(*MapType).ChanDir"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/runtime/abi.(*MapType).ChanDir"(ptr %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce ptr @"__llgo_stub.{{.*}}/runtime/abi.(*MapType).Common"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call ptr @"{{.*}}/runtime/abi.(*MapType).Common"(ptr %1)
// CHECK-NEXT:   ret ptr %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.Slice" @"__llgo_stub.{{.*}}/runtime/abi.(*MapType).ExportedMethods"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/abi.(*MapType).ExportedMethods"(ptr %1)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.Slice" %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/runtime/abi.(*MapType).FieldAlign"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/runtime/abi.(*MapType).FieldAlign"(ptr %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce ptr @"__llgo_stub.{{.*}}/runtime/abi.(*MapType).FuncType"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call ptr @"{{.*}}/runtime/abi.(*MapType).FuncType"(ptr %1)
// CHECK-NEXT:   ret ptr %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.Slice" @"__llgo_stub.{{.*}}/runtime/abi.(*MapType).GcSlice"(ptr %0, ptr %1, i64 %2, i64 %3){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %4 = tail call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/abi.(*MapType).GcSlice"(ptr %1, i64 %2, i64 %3)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.Slice" %4
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/abi.(*MapType).HasName"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i1 @"{{.*}}/runtime/abi.(*MapType).HasName"(ptr %1)
// CHECK-NEXT:   ret i1 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/abi.(*MapType).HashMightPanic"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i1 @"{{.*}}/runtime/abi.(*MapType).HashMightPanic"(ptr %1)
// CHECK-NEXT:   ret i1 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/abi.(*MapType).IfaceIndir"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i1 @"{{.*}}/runtime/abi.(*MapType).IfaceIndir"(ptr %1)
// CHECK-NEXT:   ret i1 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/abi.(*MapType).IndirectElem"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i1 @"{{.*}}/runtime/abi.(*MapType).IndirectElem"(ptr %1)
// CHECK-NEXT:   ret i1 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/abi.(*MapType).IndirectKey"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i1 @"{{.*}}/runtime/abi.(*MapType).IndirectKey"(ptr %1)
// CHECK-NEXT:   ret i1 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce ptr @"__llgo_stub.{{.*}}/runtime/abi.(*MapType).InterfaceType"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call ptr @"{{.*}}/runtime/abi.(*MapType).InterfaceType"(ptr %1)
// CHECK-NEXT:   ret ptr %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/abi.(*MapType).IsClosure"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i1 @"{{.*}}/runtime/abi.(*MapType).IsClosure"(ptr %1)
// CHECK-NEXT:   ret i1 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/abi.(*MapType).IsDirectIface"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i1 @"{{.*}}/runtime/abi.(*MapType).IsDirectIface"(ptr %1)
// CHECK-NEXT:   ret i1 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/runtime/abi.(*MapType).Kind"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/runtime/abi.(*MapType).Kind"(ptr %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/runtime/abi.(*MapType).Len"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/runtime/abi.(*MapType).Len"(ptr %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce ptr @"__llgo_stub.{{.*}}/runtime/abi.(*MapType).MapType"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call ptr @"{{.*}}/runtime/abi.(*MapType).MapType"(ptr %1)
// CHECK-NEXT:   ret ptr %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/abi.(*MapType).NeedKeyUpdate"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i1 @"{{.*}}/runtime/abi.(*MapType).NeedKeyUpdate"(ptr %1)
// CHECK-NEXT:   ret i1 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/runtime/abi.(*MapType).NumMethod"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/runtime/abi.(*MapType).NumMethod"(ptr %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/abi.(*MapType).Pointers"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i1 @"{{.*}}/runtime/abi.(*MapType).Pointers"(ptr %1)
// CHECK-NEXT:   ret i1 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/abi.(*MapType).ReflexiveKey"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i1 @"{{.*}}/runtime/abi.(*MapType).ReflexiveKey"(ptr %1)
// CHECK-NEXT:   ret i1 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/runtime/abi.(*MapType).Size"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/runtime/abi.(*MapType).Size"(ptr %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.String" @"__llgo_stub.{{.*}}/runtime/abi.(*MapType).String"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/runtime/abi.(*MapType).String"(ptr %1)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.String" %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce ptr @"__llgo_stub.{{.*}}/runtime/abi.(*MapType).StructType"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call ptr @"{{.*}}/runtime/abi.(*MapType).StructType"(ptr %1)
// CHECK-NEXT:   ret ptr %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/abi.(*StructField).Embedded"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i1 @"{{.*}}/runtime/abi.(*StructField).Embedded"(ptr %1)
// CHECK-NEXT:   ret i1 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/abi.(*StructField).Exported"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i1 @"{{.*}}/runtime/abi.(*StructField).Exported"(ptr %1)
// CHECK-NEXT:   ret i1 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/runtime/abi.(*StructType).Align"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/runtime/abi.(*StructType).Align"(ptr %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce ptr @"__llgo_stub.{{.*}}/runtime/abi.(*StructType).ArrayType"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call ptr @"{{.*}}/runtime/abi.(*StructType).ArrayType"(ptr %1)
// CHECK-NEXT:   ret ptr %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/runtime/abi.(*StructType).ChanDir"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/runtime/abi.(*StructType).ChanDir"(ptr %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce ptr @"__llgo_stub.{{.*}}/runtime/abi.(*StructType).Common"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call ptr @"{{.*}}/runtime/abi.(*StructType).Common"(ptr %1)
// CHECK-NEXT:   ret ptr %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce ptr @"__llgo_stub.{{.*}}/runtime/abi.(*StructType).Elem"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call ptr @"{{.*}}/runtime/abi.(*StructType).Elem"(ptr %1)
// CHECK-NEXT:   ret ptr %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.Slice" @"__llgo_stub.{{.*}}/runtime/abi.(*StructType).ExportedMethods"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/abi.(*StructType).ExportedMethods"(ptr %1)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.Slice" %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/runtime/abi.(*StructType).FieldAlign"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/runtime/abi.(*StructType).FieldAlign"(ptr %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce ptr @"__llgo_stub.{{.*}}/runtime/abi.(*StructType).FuncType"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call ptr @"{{.*}}/runtime/abi.(*StructType).FuncType"(ptr %1)
// CHECK-NEXT:   ret ptr %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.Slice" @"__llgo_stub.{{.*}}/runtime/abi.(*StructType).GcSlice"(ptr %0, ptr %1, i64 %2, i64 %3){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %4 = tail call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/abi.(*StructType).GcSlice"(ptr %1, i64 %2, i64 %3)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.Slice" %4
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/abi.(*StructType).HasName"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i1 @"{{.*}}/runtime/abi.(*StructType).HasName"(ptr %1)
// CHECK-NEXT:   ret i1 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/abi.(*StructType).IfaceIndir"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i1 @"{{.*}}/runtime/abi.(*StructType).IfaceIndir"(ptr %1)
// CHECK-NEXT:   ret i1 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce ptr @"__llgo_stub.{{.*}}/runtime/abi.(*StructType).InterfaceType"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call ptr @"{{.*}}/runtime/abi.(*StructType).InterfaceType"(ptr %1)
// CHECK-NEXT:   ret ptr %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/abi.(*StructType).IsClosure"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i1 @"{{.*}}/runtime/abi.(*StructType).IsClosure"(ptr %1)
// CHECK-NEXT:   ret i1 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/abi.(*StructType).IsDirectIface"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i1 @"{{.*}}/runtime/abi.(*StructType).IsDirectIface"(ptr %1)
// CHECK-NEXT:   ret i1 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce ptr @"__llgo_stub.{{.*}}/runtime/abi.(*StructType).Key"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call ptr @"{{.*}}/runtime/abi.(*StructType).Key"(ptr %1)
// CHECK-NEXT:   ret ptr %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/runtime/abi.(*StructType).Kind"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/runtime/abi.(*StructType).Kind"(ptr %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/runtime/abi.(*StructType).Len"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/runtime/abi.(*StructType).Len"(ptr %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce ptr @"__llgo_stub.{{.*}}/runtime/abi.(*StructType).MapType"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call ptr @"{{.*}}/runtime/abi.(*StructType).MapType"(ptr %1)
// CHECK-NEXT:   ret ptr %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/runtime/abi.(*StructType).NumMethod"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/runtime/abi.(*StructType).NumMethod"(ptr %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/abi.(*StructType).Pointers"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i1 @"{{.*}}/runtime/abi.(*StructType).Pointers"(ptr %1)
// CHECK-NEXT:   ret i1 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/runtime/abi.(*StructType).Size"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/runtime/abi.(*StructType).Size"(ptr %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.String" @"__llgo_stub.{{.*}}/runtime/abi.(*StructType).String"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/runtime/abi.(*StructType).String"(ptr %1)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.String" %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce ptr @"__llgo_stub.{{.*}}/runtime/abi.(*StructType).StructType"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call ptr @"{{.*}}/runtime/abi.(*StructType).StructType"(ptr %1)
// CHECK-NEXT:   ret ptr %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce ptr @"__llgo_stub.{{.*}}/runtime/abi.(*StructType).Uncommon"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call ptr @"{{.*}}/runtime/abi.(*StructType).Uncommon"(ptr %1)
// CHECK-NEXT:   ret ptr %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.Slice" @"__llgo_stub.{{.*}}/runtime/abi.(*UncommonType).ExportedMethods"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/abi.(*UncommonType).ExportedMethods"(ptr %1)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.Slice" %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.Slice" @"__llgo_stub.{{.*}}/runtime/abi.(*UncommonType).Methods"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/abi.(*UncommonType).Methods"(ptr %1)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.Slice" %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce ptr @"__llgo_stub.{{.*}}/runtime/abi.(*MapType).Uncommon"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call ptr @"{{.*}}/runtime/abi.(*MapType).Uncommon"(ptr %1)
// CHECK-NEXT:   ret ptr %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/runtime/abi.(*InterfaceType).NumMethod"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/runtime/abi.(*InterfaceType).NumMethod"(ptr %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/abi.(*InterfaceType).Pointers"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i1 @"{{.*}}/runtime/abi.(*InterfaceType).Pointers"(ptr %1)
// CHECK-NEXT:   ret i1 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/runtime/abi.(*InterfaceType).Size"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/runtime/abi.(*InterfaceType).Size"(ptr %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.String" @"__llgo_stub.{{.*}}/runtime/abi.(*InterfaceType).String"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/runtime/abi.(*InterfaceType).String"(ptr %1)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.String" %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce ptr @"__llgo_stub.{{.*}}/runtime/abi.(*InterfaceType).StructType"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call ptr @"{{.*}}/runtime/abi.(*InterfaceType).StructType"(ptr %1)
// CHECK-NEXT:   ret ptr %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce ptr @"__llgo_stub.{{.*}}/runtime/abi.(*InterfaceType).Uncommon"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call ptr @"{{.*}}/runtime/abi.(*InterfaceType).Uncommon"(ptr %1)
// CHECK-NEXT:   ret ptr %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/abi.(*FuncType).IsClosure"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i1 @"{{.*}}/runtime/abi.(*FuncType).IsClosure"(ptr %1)
// CHECK-NEXT:   ret i1 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/abi.(*FuncType).IsDirectIface"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i1 @"{{.*}}/runtime/abi.(*FuncType).IsDirectIface"(ptr %1)
// CHECK-NEXT:   ret i1 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce ptr @"__llgo_stub.{{.*}}/runtime/abi.(*FuncType).Key"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call ptr @"{{.*}}/runtime/abi.(*FuncType).Key"(ptr %1)
// CHECK-NEXT:   ret ptr %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/runtime/abi.(*FuncType).Kind"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/runtime/abi.(*FuncType).Kind"(ptr %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/runtime/abi.(*FuncType).Len"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/runtime/abi.(*FuncType).Len"(ptr %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce ptr @"__llgo_stub.{{.*}}/runtime/abi.(*FuncType).MapType"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call ptr @"{{.*}}/runtime/abi.(*FuncType).MapType"(ptr %1)
// CHECK-NEXT:   ret ptr %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/runtime/abi.(*FuncType).NumMethod"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/runtime/abi.(*FuncType).NumMethod"(ptr %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/abi.(*FuncType).Pointers"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i1 @"{{.*}}/runtime/abi.(*FuncType).Pointers"(ptr %1)
// CHECK-NEXT:   ret i1 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/runtime/abi.(*FuncType).Size"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/runtime/abi.(*FuncType).Size"(ptr %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.String" @"__llgo_stub.{{.*}}/runtime/abi.(*FuncType).String"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/runtime/abi.(*FuncType).String"(ptr %1)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.String" %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce ptr @"__llgo_stub.{{.*}}/runtime/abi.(*FuncType).StructType"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call ptr @"{{.*}}/runtime/abi.(*FuncType).StructType"(ptr %1)
// CHECK-NEXT:   ret ptr %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce ptr @"__llgo_stub.{{.*}}/runtime/abi.(*FuncType).Uncommon"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call ptr @"{{.*}}/runtime/abi.(*FuncType).Uncommon"(ptr %1)
// CHECK-NEXT:   ret ptr %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/abi.(*FuncType).Variadic"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i1 @"{{.*}}/runtime/abi.(*FuncType).Variadic"(ptr %1)
// CHECK-NEXT:   ret i1 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/runtime/abi.(*ArrayType).FieldAlign"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/runtime/abi.(*ArrayType).FieldAlign"(ptr %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce ptr @"__llgo_stub.{{.*}}/runtime/abi.(*ArrayType).FuncType"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call ptr @"{{.*}}/runtime/abi.(*ArrayType).FuncType"(ptr %1)
// CHECK-NEXT:   ret ptr %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.Slice" @"__llgo_stub.{{.*}}/runtime/abi.(*ArrayType).GcSlice"(ptr %0, ptr %1, i64 %2, i64 %3){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %4 = tail call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/abi.(*ArrayType).GcSlice"(ptr %1, i64 %2, i64 %3)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.Slice" %4
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/abi.(*ArrayType).HasName"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i1 @"{{.*}}/runtime/abi.(*ArrayType).HasName"(ptr %1)
// CHECK-NEXT:   ret i1 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/abi.(*ArrayType).IfaceIndir"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i1 @"{{.*}}/runtime/abi.(*ArrayType).IfaceIndir"(ptr %1)
// CHECK-NEXT:   ret i1 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce ptr @"__llgo_stub.{{.*}}/runtime/abi.(*ArrayType).InterfaceType"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call ptr @"{{.*}}/runtime/abi.(*ArrayType).InterfaceType"(ptr %1)
// CHECK-NEXT:   ret ptr %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/abi.(*ArrayType).IsClosure"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i1 @"{{.*}}/runtime/abi.(*ArrayType).IsClosure"(ptr %1)
// CHECK-NEXT:   ret i1 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/abi.(*ArrayType).IsDirectIface"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i1 @"{{.*}}/runtime/abi.(*ArrayType).IsDirectIface"(ptr %1)
// CHECK-NEXT:   ret i1 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce ptr @"__llgo_stub.{{.*}}/runtime/abi.(*ArrayType).Key"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call ptr @"{{.*}}/runtime/abi.(*ArrayType).Key"(ptr %1)
// CHECK-NEXT:   ret ptr %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/runtime/abi.(*ArrayType).Kind"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/runtime/abi.(*ArrayType).Kind"(ptr %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce ptr @"__llgo_stub.{{.*}}/runtime/abi.(*ArrayType).MapType"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call ptr @"{{.*}}/runtime/abi.(*ArrayType).MapType"(ptr %1)
// CHECK-NEXT:   ret ptr %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/runtime/abi.(*ArrayType).NumMethod"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/runtime/abi.(*ArrayType).NumMethod"(ptr %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/abi.(*ArrayType).Pointers"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i1 @"{{.*}}/runtime/abi.(*ArrayType).Pointers"(ptr %1)
// CHECK-NEXT:   ret i1 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/runtime/abi.(*ArrayType).Size"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/runtime/abi.(*ArrayType).Size"(ptr %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.String" @"__llgo_stub.{{.*}}/runtime/abi.(*ArrayType).String"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/runtime/abi.(*ArrayType).String"(ptr %1)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.String" %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce ptr @"__llgo_stub.{{.*}}/runtime/abi.(*ArrayType).StructType"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call ptr @"{{.*}}/runtime/abi.(*ArrayType).StructType"(ptr %1)
// CHECK-NEXT:   ret ptr %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce ptr @"__llgo_stub.{{.*}}/runtime/abi.(*ArrayType).Uncommon"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call ptr @"{{.*}}/runtime/abi.(*ArrayType).Uncommon"(ptr %1)
// CHECK-NEXT:   ret ptr %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/runtime/abi.(*Type).ChanDir"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/runtime/abi.(*Type).ChanDir"(ptr %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce ptr @"__llgo_stub.{{.*}}/runtime/abi.(*Type).Common"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call ptr @"{{.*}}/runtime/abi.(*Type).Common"(ptr %1)
// CHECK-NEXT:   ret ptr %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce ptr @"__llgo_stub.{{.*}}/runtime/abi.(*Type).Elem"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call ptr @"{{.*}}/runtime/abi.(*Type).Elem"(ptr %1)
// CHECK-NEXT:   ret ptr %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.Slice" @"__llgo_stub.{{.*}}/runtime/abi.(*Type).ExportedMethods"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/abi.(*Type).ExportedMethods"(ptr %1)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.Slice" %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/runtime/abi.(*Type).FieldAlign"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/runtime/abi.(*Type).FieldAlign"(ptr %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce ptr @"__llgo_stub.{{.*}}/runtime/abi.(*Type).FuncType"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call ptr @"{{.*}}/runtime/abi.(*Type).FuncType"(ptr %1)
// CHECK-NEXT:   ret ptr %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.Slice" @"__llgo_stub.{{.*}}/runtime/abi.(*Type).GcSlice"(ptr %0, ptr %1, i64 %2, i64 %3){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %4 = tail call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/abi.(*Type).GcSlice"(ptr %1, i64 %2, i64 %3)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.Slice" %4
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/abi.(*Type).HasName"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i1 @"{{.*}}/runtime/abi.(*Type).HasName"(ptr %1)
// CHECK-NEXT:   ret i1 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/abi.(*Type).IfaceIndir"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i1 @"{{.*}}/runtime/abi.(*Type).IfaceIndir"(ptr %1)
// CHECK-NEXT:   ret i1 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce ptr @"__llgo_stub.{{.*}}/runtime/abi.(*Type).InterfaceType"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call ptr @"{{.*}}/runtime/abi.(*Type).InterfaceType"(ptr %1)
// CHECK-NEXT:   ret ptr %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/abi.(*Type).IsClosure"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i1 @"{{.*}}/runtime/abi.(*Type).IsClosure"(ptr %1)
// CHECK-NEXT:   ret i1 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/abi.(*Type).IsDirectIface"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i1 @"{{.*}}/runtime/abi.(*Type).IsDirectIface"(ptr %1)
// CHECK-NEXT:   ret i1 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce ptr @"__llgo_stub.{{.*}}/runtime/abi.(*Type).Key"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call ptr @"{{.*}}/runtime/abi.(*Type).Key"(ptr %1)
// CHECK-NEXT:   ret ptr %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/runtime/abi.(*Type).Kind"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/runtime/abi.(*Type).Kind"(ptr %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/runtime/abi.(*Type).Len"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/runtime/abi.(*Type).Len"(ptr %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce ptr @"__llgo_stub.{{.*}}/runtime/abi.(*Type).MapType"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call ptr @"{{.*}}/runtime/abi.(*Type).MapType"(ptr %1)
// CHECK-NEXT:   ret ptr %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/runtime/abi.(*Type).NumMethod"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/runtime/abi.(*Type).NumMethod"(ptr %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/abi.(*Type).Pointers"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i1 @"{{.*}}/runtime/abi.(*Type).Pointers"(ptr %1)
// CHECK-NEXT:   ret i1 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/runtime/abi.(*Type).Size"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/runtime/abi.(*Type).Size"(ptr %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce %"{{.*}}/runtime/internal/runtime.String" @"__llgo_stub.{{.*}}/runtime/abi.(*Type).String"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/runtime/abi.(*Type).String"(ptr %1)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.String" %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce ptr @"__llgo_stub.{{.*}}/runtime/abi.(*Type).StructType"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call ptr @"{{.*}}/runtime/abi.(*Type).StructType"(ptr %1)
// CHECK-NEXT:   ret ptr %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce ptr @"__llgo_stub.{{.*}}/runtime/abi.(*Type).Uncommon"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call ptr @"{{.*}}/runtime/abi.(*Type).Uncommon"(ptr %1)
// CHECK-NEXT:   ret ptr %2
// CHECK-NEXT: }
