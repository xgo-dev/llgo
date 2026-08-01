// LITTEST
package main

import (
	"unsafe"

	"github.com/goplus/llgo/runtime/abi"
)

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 48)
// CHECK-NEXT:   store %main.T zeroinitializer, ptr %0, align 8
// CHECK-NEXT:   %1 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.T, ptr undef }, ptr %0, 1
// CHECK-NEXT:   %2 = call ptr @main.toEface(%"{{.*}}/runtime/internal/runtime.eface" %1)
// CHECK-NEXT:   %3 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 72)
// CHECK-NEXT:   store %"{{.*}}/runtime/abi.Type" zeroinitializer, ptr %3, align 8
// CHECK-NEXT:   %4 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"_llgo_{{.*}}/runtime/abi.Type", ptr undef }, ptr %3, 1
// CHECK-NEXT:   %5 = call ptr @main.toEface(%"{{.*}}/runtime/internal/runtime.eface" %4)
// CHECK-NEXT:   %6 = getelementptr inbounds %main.eface, ptr %2, i32 0, i32 0
// CHECK-NEXT:   %7 = icmp eq ptr %2, null
// CHECK-NEXT:   br i1 %7, label %36, label %37
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %107
// CHECK-NEXT:   %8 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @121, i64 13 }, ptr %8, align 8
// CHECK-NEXT:   %9 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %8, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %9)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %107
// CHECK-NEXT:   %10 = getelementptr inbounds %"{{.*}}/runtime/abi.StructField", ptr %72, i32 0, i32 1
// CHECK-NEXT:   %11 = load ptr, ptr %10, align 8
// CHECK-NEXT:   %12 = call ptr @"{{.*}}/runtime/abi.(*Type).Elem"(ptr %11)
// CHECK-NEXT:   %13 = getelementptr inbounds %main.eface, ptr %2, i32 0, i32 0
// CHECK-NEXT:   %14 = icmp eq ptr %2, null
// CHECK-NEXT:   br i1 %14, label %110, label %111
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %111
// CHECK-NEXT:   %15 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @122, i64 18 }, ptr %15, align 8
// CHECK-NEXT:   %16 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %15, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %16)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %111
// CHECK-NEXT:   %17 = alloca %"{{.*}}/runtime/abi.StructField", align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %17, i8 0, i64 56, i1 false)
// CHECK-NEXT:   %18 = icmp eq ptr %2, null
// CHECK-NEXT:   br i1 %18, label %114, label %115
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %147
// CHECK-NEXT:   %19 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @123, i64 13 }, ptr %19, align 8
// CHECK-NEXT:   %20 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %19, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %20)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %147
// CHECK-NEXT:   %21 = getelementptr inbounds %"{{.*}}/runtime/abi.StructField", ptr %17, i32 0, i32 1
// CHECK-NEXT:   %22 = load ptr, ptr %21, align 8
// CHECK-NEXT:   %23 = call ptr @"{{.*}}/runtime/abi.(*Type).Elem"(ptr %22)
// CHECK-NEXT:   %24 = getelementptr inbounds %main.eface, ptr %5, i32 0, i32 0
// CHECK-NEXT:   %25 = icmp eq ptr %5, null
// CHECK-NEXT:   br i1 %25, label %150, label %151
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_7:                                          ; preds = %151
// CHECK-NEXT:   %26 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @124, i64 18 }, ptr %26, align 8
// CHECK-NEXT:   %27 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %26, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %27)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_8:                                          ; preds = %151
// CHECK-NEXT:   %28 = alloca %"{{.*}}/runtime/abi.StructField", align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %28, i8 0, i64 56, i1 false)
// CHECK-NEXT:   %29 = icmp eq ptr %2, null
// CHECK-NEXT:   br i1 %29, label %154, label %155
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_9:                                          ; preds = %197
// CHECK-NEXT:   %30 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @125, i64 13 }, ptr %30, align 8
// CHECK-NEXT:   %31 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %30, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %31)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_10:                                         ; preds = %197
// CHECK-NEXT:   %32 = alloca %"{{.*}}/runtime/abi.StructField", align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %32, i8 0, i64 56, i1 false)
// CHECK-NEXT:   %33 = icmp eq ptr %2, null
// CHECK-NEXT:   br i1 %33, label %200, label %201
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_11:                                         ; preds = %225
// CHECK-NEXT:   %34 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @126, i64 13 }, ptr %34, align 8
// CHECK-NEXT:   %35 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %34, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %35)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_12:                                         ; preds = %225
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: 36:                                               ; preds = %_llgo_0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 37:                                               ; preds = %_llgo_0
// CHECK-NEXT:   %38 = load ptr, ptr %6, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintPointer"(ptr %38)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %39 = getelementptr inbounds %main.eface, ptr %2, i32 0, i32 0
// CHECK-NEXT:   %40 = icmp eq ptr %2, null
// CHECK-NEXT:   br i1 %40, label %41, label %42
// CHECK-EMPTY:
// CHECK-NEXT: 41:                                               ; preds = %37
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 42:                                               ; preds = %37
// CHECK-NEXT:   %43 = load ptr, ptr %39, align 8
// CHECK-NEXT:   %44 = getelementptr inbounds %"{{.*}}/runtime/abi.Type", ptr %43, i32 0, i32 10
// CHECK-NEXT:   %45 = icmp eq ptr %2, null
// CHECK-NEXT:   br i1 %45, label %46, label %47
// CHECK-EMPTY:
// CHECK-NEXT: 46:                                               ; preds = %42
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 47:                                               ; preds = %42
// CHECK-NEXT:   %48 = icmp eq ptr %39, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %48)
// CHECK-NEXT:   %49 = icmp eq ptr %43, null
// CHECK-NEXT:   br i1 %49, label %50, label %51
// CHECK-EMPTY:
// CHECK-NEXT: 50:                                               ; preds = %47
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 51:                                               ; preds = %47
// CHECK-NEXT:   %52 = load ptr, ptr %44, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintPointer"(ptr %52)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %53 = getelementptr inbounds %main.eface, ptr %5, i32 0, i32 0
// CHECK-NEXT:   %54 = icmp eq ptr %5, null
// CHECK-NEXT:   br i1 %54, label %55, label %56
// CHECK-EMPTY:
// CHECK-NEXT: 55:                                               ; preds = %51
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 56:                                               ; preds = %51
// CHECK-NEXT:   %57 = load ptr, ptr %53, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintPointer"(ptr %57)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %58 = getelementptr inbounds %main.eface, ptr %5, i32 0, i32 0
// CHECK-NEXT:   %59 = icmp eq ptr %5, null
// CHECK-NEXT:   br i1 %59, label %60, label %61
// CHECK-EMPTY:
// CHECK-NEXT: 60:                                               ; preds = %56
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 61:                                               ; preds = %56
// CHECK-NEXT:   %62 = load ptr, ptr %58, align 8
// CHECK-NEXT:   %63 = getelementptr inbounds %"{{.*}}/runtime/abi.Type", ptr %62, i32 0, i32 10
// CHECK-NEXT:   %64 = icmp eq ptr %5, null
// CHECK-NEXT:   br i1 %64, label %65, label %66
// CHECK-EMPTY:
// CHECK-NEXT: 65:                                               ; preds = %61
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 66:                                               ; preds = %61
// CHECK-NEXT:   %67 = icmp eq ptr %58, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %67)
// CHECK-NEXT:   %68 = icmp eq ptr %62, null
// CHECK-NEXT:   br i1 %68, label %69, label %70
// CHECK-EMPTY:
// CHECK-NEXT: 69:                                               ; preds = %66
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 70:                                               ; preds = %66
// CHECK-NEXT:   %71 = load ptr, ptr %63, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintPointer"(ptr %71)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %72 = alloca %"{{.*}}/runtime/abi.StructField", align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %72, i8 0, i64 56, i1 false)
// CHECK-NEXT:   %73 = icmp eq ptr %2, null
// CHECK-NEXT:   br i1 %73, label %74, label %75
// CHECK-EMPTY:
// CHECK-NEXT: 74:                                               ; preds = %70
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 75:                                               ; preds = %70
// CHECK-NEXT:   %76 = getelementptr inbounds %main.eface, ptr %2, i32 0, i32 0
// CHECK-NEXT:   %77 = load ptr, ptr %76, align 8
// CHECK-NEXT:   %78 = call ptr @"{{.*}}/runtime/abi.(*Type).StructType"(ptr %77)
// CHECK-NEXT:   %79 = getelementptr inbounds %"{{.*}}/runtime/abi.StructType", ptr %78, i32 0, i32 2
// CHECK-NEXT:   %80 = icmp eq ptr %78, null
// CHECK-NEXT:   br i1 %80, label %81, label %82
// CHECK-EMPTY:
// CHECK-NEXT: 81:                                               ; preds = %75
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 82:                                               ; preds = %75
// CHECK-NEXT:   %83 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %79, align 8
// CHECK-NEXT:   %84 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %83, 0
// CHECK-NEXT:   %85 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %83, 1
// CHECK-NEXT:   %86 = icmp uge i64 0, %85
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.CheckIndexRange"(i1 %86, i64 0, i1 true, i64 %85)
// CHECK-NEXT:   %87 = getelementptr inbounds %"{{.*}}/runtime/abi.StructField", ptr %84, i64 0
// CHECK-NEXT:   %88 = icmp eq ptr %78, null
// CHECK-NEXT:   br i1 %88, label %89, label %90
// CHECK-EMPTY:
// CHECK-NEXT: 89:                                               ; preds = %82
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 90:                                               ; preds = %82
// CHECK-NEXT:   %91 = icmp eq ptr %79, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %91)
// CHECK-NEXT:   %92 = load %"{{.*}}/runtime/abi.StructField", ptr %87, align 8
// CHECK-NEXT:   store %"{{.*}}/runtime/abi.StructField" %92, ptr %72, align 8
// CHECK-NEXT:   %93 = getelementptr inbounds %"{{.*}}/runtime/abi.StructField", ptr %72, i32 0, i32 1
// CHECK-NEXT:   %94 = load ptr, ptr %93, align 8
// CHECK-NEXT:   %95 = getelementptr inbounds %main.eface, ptr %2, i32 0, i32 0
// CHECK-NEXT:   %96 = icmp eq ptr %2, null
// CHECK-NEXT:   br i1 %96, label %97, label %98
// CHECK-EMPTY:
// CHECK-NEXT: 97:                                               ; preds = %90
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 98:                                               ; preds = %90
// CHECK-NEXT:   %99 = load ptr, ptr %95, align 8
// CHECK-NEXT:   %100 = getelementptr inbounds %"{{.*}}/runtime/abi.Type", ptr %99, i32 0, i32 10
// CHECK-NEXT:   %101 = icmp eq ptr %2, null
// CHECK-NEXT:   br i1 %101, label %102, label %103
// CHECK-EMPTY:
// CHECK-NEXT: 102:                                              ; preds = %98
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 103:                                              ; preds = %98
// CHECK-NEXT:   %104 = icmp eq ptr %95, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %104)
// CHECK-NEXT:   %105 = icmp eq ptr %99, null
// CHECK-NEXT:   br i1 %105, label %106, label %107
// CHECK-EMPTY:
// CHECK-NEXT: 106:                                              ; preds = %103
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 107:                                              ; preds = %103
// CHECK-NEXT:   %108 = load ptr, ptr %100, align 8
// CHECK-NEXT:   %109 = icmp ne ptr %94, %108
// CHECK-NEXT:   br i1 %109, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: 110:                                              ; preds = %_llgo_2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 111:                                              ; preds = %_llgo_2
// CHECK-NEXT:   %112 = load ptr, ptr %13, align 8
// CHECK-NEXT:   %113 = icmp ne ptr %12, %112
// CHECK-NEXT:   br i1 %113, label %_llgo_3, label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: 114:                                              ; preds = %_llgo_4
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 115:                                              ; preds = %_llgo_4
// CHECK-NEXT:   %116 = getelementptr inbounds %main.eface, ptr %2, i32 0, i32 0
// CHECK-NEXT:   %117 = load ptr, ptr %116, align 8
// CHECK-NEXT:   %118 = call ptr @"{{.*}}/runtime/abi.(*Type).StructType"(ptr %117)
// CHECK-NEXT:   %119 = getelementptr inbounds %"{{.*}}/runtime/abi.StructType", ptr %118, i32 0, i32 2
// CHECK-NEXT:   %120 = icmp eq ptr %118, null
// CHECK-NEXT:   br i1 %120, label %121, label %122
// CHECK-EMPTY:
// CHECK-NEXT: 121:                                              ; preds = %115
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 122:                                              ; preds = %115
// CHECK-NEXT:   %123 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %119, align 8
// CHECK-NEXT:   %124 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %123, 0
// CHECK-NEXT:   %125 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %123, 1
// CHECK-NEXT:   %126 = icmp uge i64 1, %125
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.CheckIndexRange"(i1 %126, i64 1, i1 true, i64 %125)
// CHECK-NEXT:   %127 = getelementptr inbounds %"{{.*}}/runtime/abi.StructField", ptr %124, i64 1
// CHECK-NEXT:   %128 = icmp eq ptr %118, null
// CHECK-NEXT:   br i1 %128, label %129, label %130
// CHECK-EMPTY:
// CHECK-NEXT: 129:                                              ; preds = %122
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 130:                                              ; preds = %122
// CHECK-NEXT:   %131 = icmp eq ptr %119, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %131)
// CHECK-NEXT:   %132 = load %"{{.*}}/runtime/abi.StructField", ptr %127, align 8
// CHECK-NEXT:   store %"{{.*}}/runtime/abi.StructField" %132, ptr %17, align 8
// CHECK-NEXT:   %133 = getelementptr inbounds %"{{.*}}/runtime/abi.StructField", ptr %17, i32 0, i32 1
// CHECK-NEXT:   %134 = load ptr, ptr %133, align 8
// CHECK-NEXT:   %135 = getelementptr inbounds %main.eface, ptr %5, i32 0, i32 0
// CHECK-NEXT:   %136 = icmp eq ptr %5, null
// CHECK-NEXT:   br i1 %136, label %137, label %138
// CHECK-EMPTY:
// CHECK-NEXT: 137:                                              ; preds = %130
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 138:                                              ; preds = %130
// CHECK-NEXT:   %139 = load ptr, ptr %135, align 8
// CHECK-NEXT:   %140 = getelementptr inbounds %"{{.*}}/runtime/abi.Type", ptr %139, i32 0, i32 10
// CHECK-NEXT:   %141 = icmp eq ptr %5, null
// CHECK-NEXT:   br i1 %141, label %142, label %143
// CHECK-EMPTY:
// CHECK-NEXT: 142:                                              ; preds = %138
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 143:                                              ; preds = %138
// CHECK-NEXT:   %144 = icmp eq ptr %135, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %144)
// CHECK-NEXT:   %145 = icmp eq ptr %139, null
// CHECK-NEXT:   br i1 %145, label %146, label %147
// CHECK-EMPTY:
// CHECK-NEXT: 146:                                              ; preds = %143
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 147:                                              ; preds = %143
// CHECK-NEXT:   %148 = load ptr, ptr %140, align 8
// CHECK-NEXT:   %149 = icmp ne ptr %134, %148
// CHECK-NEXT:   br i1 %149, label %_llgo_5, label %_llgo_6
// CHECK-EMPTY:
// CHECK-NEXT: 150:                                              ; preds = %_llgo_6
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 151:                                              ; preds = %_llgo_6
// CHECK-NEXT:   %152 = load ptr, ptr %24, align 8
// CHECK-NEXT:   %153 = icmp ne ptr %23, %152
// CHECK-NEXT:   br i1 %153, label %_llgo_7, label %_llgo_8
// CHECK-EMPTY:
// CHECK-NEXT: 154:                                              ; preds = %_llgo_8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 155:                                              ; preds = %_llgo_8
// CHECK-NEXT:   %156 = getelementptr inbounds %main.eface, ptr %2, i32 0, i32 0
// CHECK-NEXT:   %157 = load ptr, ptr %156, align 8
// CHECK-NEXT:   %158 = call ptr @"{{.*}}/runtime/abi.(*Type).StructType"(ptr %157)
// CHECK-NEXT:   %159 = getelementptr inbounds %"{{.*}}/runtime/abi.StructType", ptr %158, i32 0, i32 2
// CHECK-NEXT:   %160 = icmp eq ptr %158, null
// CHECK-NEXT:   br i1 %160, label %161, label %162
// CHECK-EMPTY:
// CHECK-NEXT: 161:                                              ; preds = %155
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 162:                                              ; preds = %155
// CHECK-NEXT:   %163 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %159, align 8
// CHECK-NEXT:   %164 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %163, 0
// CHECK-NEXT:   %165 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %163, 1
// CHECK-NEXT:   %166 = icmp uge i64 2, %165
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.CheckIndexRange"(i1 %166, i64 2, i1 true, i64 %165)
// CHECK-NEXT:   %167 = getelementptr inbounds %"{{.*}}/runtime/abi.StructField", ptr %164, i64 2
// CHECK-NEXT:   %168 = icmp eq ptr %158, null
// CHECK-NEXT:   br i1 %168, label %169, label %170
// CHECK-EMPTY:
// CHECK-NEXT: 169:                                              ; preds = %162
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 170:                                              ; preds = %162
// CHECK-NEXT:   %171 = icmp eq ptr %159, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %171)
// CHECK-NEXT:   %172 = load %"{{.*}}/runtime/abi.StructField", ptr %167, align 8
// CHECK-NEXT:   store %"{{.*}}/runtime/abi.StructField" %172, ptr %28, align 8
// CHECK-NEXT:   %173 = getelementptr inbounds %"{{.*}}/runtime/abi.StructField", ptr %28, i32 0, i32 1
// CHECK-NEXT:   %174 = load ptr, ptr %173, align 8
// CHECK-NEXT:   %175 = icmp eq ptr %5, null
// CHECK-NEXT:   br i1 %175, label %176, label %177
// CHECK-EMPTY:
// CHECK-NEXT: 176:                                              ; preds = %170
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 177:                                              ; preds = %170
// CHECK-NEXT:   %178 = getelementptr inbounds %main.eface, ptr %5, i32 0, i32 0
// CHECK-NEXT:   %179 = load ptr, ptr %178, align 8
// CHECK-NEXT:   %180 = call ptr @"{{.*}}/runtime/abi.(*Type).StructType"(ptr %179)
// CHECK-NEXT:   %181 = getelementptr inbounds %"{{.*}}/runtime/abi.StructType", ptr %180, i32 0, i32 2
// CHECK-NEXT:   %182 = icmp eq ptr %180, null
// CHECK-NEXT:   br i1 %182, label %183, label %184
// CHECK-EMPTY:
// CHECK-NEXT: 183:                                              ; preds = %177
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 184:                                              ; preds = %177
// CHECK-NEXT:   %185 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %181, align 8
// CHECK-NEXT:   %186 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %185, 0
// CHECK-NEXT:   %187 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %185, 1
// CHECK-NEXT:   %188 = icmp uge i64 0, %187
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.CheckIndexRange"(i1 %188, i64 0, i1 true, i64 %187)
// CHECK-NEXT:   %189 = getelementptr inbounds %"{{.*}}/runtime/abi.StructField", ptr %186, i64 0
// CHECK-NEXT:   %190 = getelementptr inbounds %"{{.*}}/runtime/abi.StructField", ptr %189, i32 0, i32 1
// CHECK-NEXT:   %191 = icmp eq ptr %180, null
// CHECK-NEXT:   br i1 %191, label %192, label %193
// CHECK-EMPTY:
// CHECK-NEXT: 192:                                              ; preds = %184
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 193:                                              ; preds = %184
// CHECK-NEXT:   %194 = icmp eq ptr %181, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %194)
// CHECK-NEXT:   %195 = icmp eq ptr %189, null
// CHECK-NEXT:   br i1 %195, label %196, label %197
// CHECK-EMPTY:
// CHECK-NEXT: 196:                                              ; preds = %193
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 197:                                              ; preds = %193
// CHECK-NEXT:   %198 = load ptr, ptr %190, align 8
// CHECK-NEXT:   %199 = icmp ne ptr %174, %198
// CHECK-NEXT:   br i1 %199, label %_llgo_9, label %_llgo_10
// CHECK-EMPTY:
// CHECK-NEXT: 200:                                              ; preds = %_llgo_10
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 201:                                              ; preds = %_llgo_10
// CHECK-NEXT:   %202 = getelementptr inbounds %main.eface, ptr %2, i32 0, i32 0
// CHECK-NEXT:   %203 = load ptr, ptr %202, align 8
// CHECK-NEXT:   %204 = call ptr @"{{.*}}/runtime/abi.(*Type).StructType"(ptr %203)
// CHECK-NEXT:   %205 = getelementptr inbounds %"{{.*}}/runtime/abi.StructType", ptr %204, i32 0, i32 2
// CHECK-NEXT:   %206 = icmp eq ptr %204, null
// CHECK-NEXT:   br i1 %206, label %207, label %208
// CHECK-EMPTY:
// CHECK-NEXT: 207:                                              ; preds = %201
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 208:                                              ; preds = %201
// CHECK-NEXT:   %209 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %205, align 8
// CHECK-NEXT:   %210 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %209, 0
// CHECK-NEXT:   %211 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %209, 1
// CHECK-NEXT:   %212 = icmp uge i64 3, %211
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.CheckIndexRange"(i1 %212, i64 3, i1 true, i64 %211)
// CHECK-NEXT:   %213 = getelementptr inbounds %"{{.*}}/runtime/abi.StructField", ptr %210, i64 3
// CHECK-NEXT:   %214 = icmp eq ptr %204, null
// CHECK-NEXT:   br i1 %214, label %215, label %216
// CHECK-EMPTY:
// CHECK-NEXT: 215:                                              ; preds = %208
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 216:                                              ; preds = %208
// CHECK-NEXT:   %217 = icmp eq ptr %205, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %217)
// CHECK-NEXT:   %218 = load %"{{.*}}/runtime/abi.StructField", ptr %213, align 8
// CHECK-NEXT:   store %"{{.*}}/runtime/abi.StructField" %218, ptr %32, align 8
// CHECK-NEXT:   %219 = getelementptr inbounds %"{{.*}}/runtime/abi.StructField", ptr %32, i32 0, i32 1
// CHECK-NEXT:   %220 = load ptr, ptr %219, align 8
// CHECK-NEXT:   %221 = call ptr @"{{.*}}/runtime/abi.(*Type).Elem"(ptr %220)
// CHECK-NEXT:   %222 = getelementptr inbounds %main.eface, ptr %2, i32 0, i32 0
// CHECK-NEXT:   %223 = icmp eq ptr %2, null
// CHECK-NEXT:   br i1 %223, label %224, label %225
// CHECK-EMPTY:
// CHECK-NEXT: 224:                                              ; preds = %216
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 225:                                              ; preds = %216
// CHECK-NEXT:   %226 = load ptr, ptr %222, align 8
// CHECK-NEXT:   %227 = icmp ne ptr %221, %226
// CHECK-NEXT:   br i1 %227, label %_llgo_11, label %_llgo_12
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

// CHECK-LABEL: define ptr @main.toEface(%"{{.*}}/runtime/internal/runtime.eface" %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %0, ptr %1, align 8
// CHECK-NEXT:   ret ptr %1
// CHECK-NEXT: }
func toEface(i any) *eface {
	return (*eface)(unsafe.Pointer(&i))
}
