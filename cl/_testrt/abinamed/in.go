// LITTEST
package main

import (
	"unsafe"

	"github.com/goplus/llgo/runtime/abi"
)

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/abinamed.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %{{[0-9]+}} = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 48)
// CHECK-NEXT:   store %"{{.*}}/cl/_testrt/abinamed.T" zeroinitializer, ptr %{{[0-9]+}}, align 8
// CHECK-NEXT:   %{{[0-9]+}} = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"_llgo_{{.*}}/cl/_testrt/abinamed.T", ptr undef }, ptr %{{[0-9]+}}, 1
// CHECK-NEXT:   %{{[0-9]+}} = call ptr @"{{.*}}/cl/_testrt/abinamed.toEface"(%"{{.*}}/runtime/internal/runtime.eface" %{{[0-9]+}})
// CHECK-NEXT:   %{{[0-9]+}} = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 72)
// CHECK-NEXT:   store %"{{.*}}/runtime/abi.Type" zeroinitializer, ptr %{{[0-9]+}}, align 8
// CHECK-NEXT:   %{{[0-9]+}} = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"_llgo_{{.*}}/runtime/abi.Type", ptr undef }, ptr %{{[0-9]+}}, 1
// CHECK-NEXT:   %{{[0-9]+}} = call ptr @"{{.*}}/cl/_testrt/abinamed.toEface"(%"{{.*}}/runtime/internal/runtime.eface" %{{[0-9]+}})
// CHECK-NEXT:   %{{[0-9]+}} = getelementptr inbounds %"{{.*}}/cl/_testrt/abinamed.eface", ptr %{{[0-9]+}}, i32 0, i32 0
// CHECK-NEXT:   %{{[0-9]+}} = load ptr, ptr %{{[0-9]+}}, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintPointer"(ptr %{{[0-9]+}})
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %{{[0-9]+}} = getelementptr inbounds %"{{.*}}/cl/_testrt/abinamed.eface", ptr %{{[0-9]+}}, i32 0, i32 0
// CHECK-NEXT:   %{{[0-9]+}} = load ptr, ptr %{{[0-9]+}}, align 8
// CHECK-NEXT:   %{{[0-9]+}} = icmp eq ptr %{{[0-9]+}}, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %{{[0-9]+}})
// CHECK-NEXT:   %{{[0-9]+}} = getelementptr inbounds %"{{.*}}/runtime/abi.Type", ptr %{{[0-9]+}}, i32 0, i32 10
// CHECK-NEXT:   %{{[0-9]+}} = load ptr, ptr %{{[0-9]+}}, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintPointer"(ptr %{{[0-9]+}})
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %{{[0-9]+}} = getelementptr inbounds %"{{.*}}/cl/_testrt/abinamed.eface", ptr %{{[0-9]+}}, i32 0, i32 0
// CHECK-NEXT:   %{{[0-9]+}} = load ptr, ptr %{{[0-9]+}}, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintPointer"(ptr %{{[0-9]+}})
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %{{[0-9]+}} = getelementptr inbounds %"{{.*}}/cl/_testrt/abinamed.eface", ptr %{{[0-9]+}}, i32 0, i32 0
// CHECK-NEXT:   %{{[0-9]+}} = load ptr, ptr %{{[0-9]+}}, align 8
// CHECK-NEXT:   %{{[0-9]+}} = icmp eq ptr %{{[0-9]+}}, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %{{[0-9]+}})
// CHECK-NEXT:   %{{[0-9]+}} = getelementptr inbounds %"{{.*}}/runtime/abi.Type", ptr %{{[0-9]+}}, i32 0, i32 10
// CHECK-NEXT:   %{{[0-9]+}} = load ptr, ptr %{{[0-9]+}}, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintPointer"(ptr %{{[0-9]+}})
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %{{[0-9]+}} = alloca %"{{.*}}/runtime/abi.StructField", align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %{{[0-9]+}}, i8 0, i64 56, i1 false)
// CHECK-NEXT:   %{{[0-9]+}} = getelementptr inbounds %"{{.*}}/cl/_testrt/abinamed.eface", ptr %{{[0-9]+}}, i32 0, i32 0
// CHECK-NEXT:   %{{[0-9]+}} = load ptr, ptr %{{[0-9]+}}, align 8
// CHECK-NEXT:   %{{[0-9]+}} = call ptr @"{{.*}}/runtime/abi.(*Type).StructType"(ptr %{{[0-9]+}})
// CHECK-NEXT:   %{{[0-9]+}} = getelementptr inbounds %"{{.*}}/runtime/abi.StructType", ptr %{{[0-9]+}}, i32 0, i32 2
// CHECK-NEXT:   %{{[0-9]+}} = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %{{[0-9]+}}, align 8
// CHECK-NEXT:   %{{[0-9]+}} = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %{{[0-9]+}}, 0
// CHECK-NEXT:   %{{[0-9]+}} = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %{{[0-9]+}}, 1
// CHECK-NEXT:   %{{[0-9]+}} = icmp uge i64 0, %{{[0-9]+}}
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertIndexRange"(i1 %{{[0-9]+}})
// CHECK-NEXT:   %{{[0-9]+}} = getelementptr inbounds %"{{.*}}/runtime/abi.StructField", ptr %{{[0-9]+}}, i64 0
// CHECK-NEXT:   %{{[0-9]+}} = load %"{{.*}}/runtime/abi.StructField", ptr %{{[0-9]+}}, align 8
// CHECK-NEXT:   store %"{{.*}}/runtime/abi.StructField" %{{[0-9]+}}, ptr %{{[0-9]+}}, align 8
// CHECK-NEXT:   %{{[0-9]+}} = getelementptr inbounds %"{{.*}}/runtime/abi.StructField", ptr %{{[0-9]+}}, i32 0, i32 1
// CHECK-NEXT:   %{{[0-9]+}} = load ptr, ptr %{{[0-9]+}}, align 8
// CHECK-NEXT:   %{{[0-9]+}} = getelementptr inbounds %"{{.*}}/cl/_testrt/abinamed.eface", ptr %{{[0-9]+}}, i32 0, i32 0
// CHECK-NEXT:   %{{[0-9]+}} = load ptr, ptr %{{[0-9]+}}, align 8
// CHECK-NEXT:   %{{[0-9]+}} = getelementptr inbounds %"{{.*}}/runtime/abi.Type", ptr %{{[0-9]+}}, i32 0, i32 10
// CHECK-NEXT:   %{{[0-9]+}} = load ptr, ptr %{{[0-9]+}}, align 8
// CHECK-NEXT:   %{{[0-9]+}} = icmp ne ptr %{{[0-9]+}}, %{{[0-9]+}}
// CHECK-NEXT:   br i1 %{{[0-9]+}}, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %{{[0-9]+}} = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{.*}}, i64 13 }, ptr %{{[0-9]+}}, align 8
// CHECK-NEXT:   %{{[0-9]+}} = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %{{[0-9]+}}, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %{{[0-9]+}})
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %{{[0-9]+}} = getelementptr inbounds %"{{.*}}/runtime/abi.StructField", ptr %{{[0-9]+}}, i32 0, i32 1
// CHECK-NEXT:   %{{[0-9]+}} = load ptr, ptr %{{[0-9]+}}, align 8
// CHECK-NEXT:   %{{[0-9]+}} = call ptr @"{{.*}}/runtime/abi.(*Type).Elem"(ptr %{{[0-9]+}})
// CHECK-NEXT:   %{{[0-9]+}} = getelementptr inbounds %"{{.*}}/cl/_testrt/abinamed.eface", ptr %{{[0-9]+}}, i32 0, i32 0
// CHECK-NEXT:   %{{[0-9]+}} = load ptr, ptr %{{[0-9]+}}, align 8
// CHECK-NEXT:   %{{[0-9]+}} = icmp ne ptr %{{[0-9]+}}, %{{[0-9]+}}
// CHECK-NEXT:   br i1 %{{[0-9]+}}, label %_llgo_3, label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %{{[0-9]+}} = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{.*}}, i64 18 }, ptr %{{[0-9]+}}, align 8
// CHECK-NEXT:   %{{[0-9]+}} = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %{{[0-9]+}}, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %{{[0-9]+}})
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %{{[0-9]+}} = alloca %"{{.*}}/runtime/abi.StructField", align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %{{[0-9]+}}, i8 0, i64 56, i1 false)
// CHECK-NEXT:   %{{[0-9]+}} = getelementptr inbounds %"{{.*}}/cl/_testrt/abinamed.eface", ptr %{{[0-9]+}}, i32 0, i32 0
// CHECK-NEXT:   %{{[0-9]+}} = load ptr, ptr %{{[0-9]+}}, align 8
// CHECK-NEXT:   %{{[0-9]+}} = call ptr @"{{.*}}/runtime/abi.(*Type).StructType"(ptr %{{[0-9]+}})
// CHECK-NEXT:   %{{[0-9]+}} = getelementptr inbounds %"{{.*}}/runtime/abi.StructType", ptr %{{[0-9]+}}, i32 0, i32 2
// CHECK-NEXT:   %{{[0-9]+}} = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %{{[0-9]+}}, align 8
// CHECK-NEXT:   %{{[0-9]+}} = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %{{[0-9]+}}, 0
// CHECK-NEXT:   %{{[0-9]+}} = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %{{[0-9]+}}, 1
// CHECK-NEXT:   %{{[0-9]+}} = icmp uge i64 1, %{{[0-9]+}}
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertIndexRange"(i1 %{{[0-9]+}})
// CHECK-NEXT:   %{{[0-9]+}} = getelementptr inbounds %"{{.*}}/runtime/abi.StructField", ptr %{{[0-9]+}}, i64 1
// CHECK-NEXT:   %{{[0-9]+}} = load %"{{.*}}/runtime/abi.StructField", ptr %{{[0-9]+}}, align 8
// CHECK-NEXT:   store %"{{.*}}/runtime/abi.StructField" %{{[0-9]+}}, ptr %{{[0-9]+}}, align 8
// CHECK-NEXT:   %{{[0-9]+}} = getelementptr inbounds %"{{.*}}/runtime/abi.StructField", ptr %{{[0-9]+}}, i32 0, i32 1
// CHECK-NEXT:   %{{[0-9]+}} = load ptr, ptr %{{[0-9]+}}, align 8
// CHECK-NEXT:   %{{[0-9]+}} = getelementptr inbounds %"{{.*}}/cl/_testrt/abinamed.eface", ptr %{{[0-9]+}}, i32 0, i32 0
// CHECK-NEXT:   %{{[0-9]+}} = load ptr, ptr %{{[0-9]+}}, align 8
// CHECK-NEXT:   %{{[0-9]+}} = getelementptr inbounds %"{{.*}}/runtime/abi.Type", ptr %{{[0-9]+}}, i32 0, i32 10
// CHECK-NEXT:   %{{[0-9]+}} = load ptr, ptr %{{[0-9]+}}, align 8
// CHECK-NEXT:   %{{[0-9]+}} = icmp ne ptr %{{[0-9]+}}, %{{[0-9]+}}
// CHECK-NEXT:   br i1 %{{[0-9]+}}, label %_llgo_5, label %_llgo_6
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_4
// CHECK-NEXT:   %{{[0-9]+}} = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{.*}}, i64 13 }, ptr %{{[0-9]+}}, align 8
// CHECK-NEXT:   %{{[0-9]+}} = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %{{[0-9]+}}, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %{{[0-9]+}})
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_4
// CHECK-NEXT:   %{{[0-9]+}} = getelementptr inbounds %"{{.*}}/runtime/abi.StructField", ptr %{{[0-9]+}}, i32 0, i32 1
// CHECK-NEXT:   %{{[0-9]+}} = load ptr, ptr %{{[0-9]+}}, align 8
// CHECK-NEXT:   %{{[0-9]+}} = call ptr @"{{.*}}/runtime/abi.(*Type).Elem"(ptr %{{[0-9]+}})
// CHECK-NEXT:   %{{[0-9]+}} = getelementptr inbounds %"{{.*}}/cl/_testrt/abinamed.eface", ptr %{{[0-9]+}}, i32 0, i32 0
// CHECK-NEXT:   %{{[0-9]+}} = load ptr, ptr %{{[0-9]+}}, align 8
// CHECK-NEXT:   %{{[0-9]+}} = icmp ne ptr %{{[0-9]+}}, %{{[0-9]+}}
// CHECK-NEXT:   br i1 %{{[0-9]+}}, label %_llgo_7, label %_llgo_8
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_7:                                          ; preds = %_llgo_6
// CHECK-NEXT:   %{{[0-9]+}} = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{.*}}, i64 18 }, ptr %{{[0-9]+}}, align 8
// CHECK-NEXT:   %{{[0-9]+}} = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %{{[0-9]+}}, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %{{[0-9]+}})
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_8:                                          ; preds = %_llgo_6
// CHECK-NEXT:   %{{[0-9]+}} = alloca %"{{.*}}/runtime/abi.StructField", align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %{{[0-9]+}}, i8 0, i64 56, i1 false)
// CHECK-NEXT:   %{{[0-9]+}} = getelementptr inbounds %"{{.*}}/cl/_testrt/abinamed.eface", ptr %{{[0-9]+}}, i32 0, i32 0
// CHECK-NEXT:   %{{[0-9]+}} = load ptr, ptr %{{[0-9]+}}, align 8
// CHECK-NEXT:   %{{[0-9]+}} = call ptr @"{{.*}}/runtime/abi.(*Type).StructType"(ptr %{{[0-9]+}})
// CHECK-NEXT:   %{{[0-9]+}} = getelementptr inbounds %"{{.*}}/runtime/abi.StructType", ptr %{{[0-9]+}}, i32 0, i32 2
// CHECK-NEXT:   %{{[0-9]+}} = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %{{[0-9]+}}, align 8
// CHECK-NEXT:   %{{[0-9]+}} = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %{{[0-9]+}}, 0
// CHECK-NEXT:   %{{[0-9]+}} = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %{{[0-9]+}}, 1
// CHECK-NEXT:   %{{[0-9]+}} = icmp uge i64 2, %{{[0-9]+}}
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertIndexRange"(i1 %{{[0-9]+}})
// CHECK-NEXT:   %{{[0-9]+}} = getelementptr inbounds %"{{.*}}/runtime/abi.StructField", ptr %{{[0-9]+}}, i64 2
// CHECK-NEXT:   %{{[0-9]+}} = load %"{{.*}}/runtime/abi.StructField", ptr %{{[0-9]+}}, align 8
// CHECK-NEXT:   store %"{{.*}}/runtime/abi.StructField" %{{[0-9]+}}, ptr %{{[0-9]+}}, align 8
// CHECK-NEXT:   %{{[0-9]+}} = getelementptr inbounds %"{{.*}}/runtime/abi.StructField", ptr %{{[0-9]+}}, i32 0, i32 1
// CHECK-NEXT:   %{{[0-9]+}} = load ptr, ptr %{{[0-9]+}}, align 8
// CHECK-NEXT:   %{{[0-9]+}} = getelementptr inbounds %"{{.*}}/cl/_testrt/abinamed.eface", ptr %{{[0-9]+}}, i32 0, i32 0
// CHECK-NEXT:   %{{[0-9]+}} = load ptr, ptr %{{[0-9]+}}, align 8
// CHECK-NEXT:   %{{[0-9]+}} = call ptr @"{{.*}}/runtime/abi.(*Type).StructType"(ptr %{{[0-9]+}})
// CHECK-NEXT:   %{{[0-9]+}} = getelementptr inbounds %"{{.*}}/runtime/abi.StructType", ptr %{{[0-9]+}}, i32 0, i32 2
// CHECK-NEXT:   %{{[0-9]+}} = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %{{[0-9]+}}, align 8
// CHECK-NEXT:   %{{[0-9]+}} = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %{{[0-9]+}}, 0
// CHECK-NEXT:   %{{[0-9]+}} = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %{{[0-9]+}}, 1
// CHECK-NEXT:   %{{[0-9]+}} = icmp uge i64 0, %{{[0-9]+}}
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertIndexRange"(i1 %{{[0-9]+}})
// CHECK-NEXT:   %{{[0-9]+}} = getelementptr inbounds %"{{.*}}/runtime/abi.StructField", ptr %{{[0-9]+}}, i64 0
// CHECK-NEXT:   %{{[0-9]+}} = getelementptr inbounds %"{{.*}}/runtime/abi.StructField", ptr %{{[0-9]+}}, i32 0, i32 1
// CHECK-NEXT:   %{{[0-9]+}} = load ptr, ptr %{{[0-9]+}}, align 8
// CHECK-NEXT:   %{{[0-9]+}} = icmp ne ptr %{{[0-9]+}}, %{{[0-9]+}}
// CHECK-NEXT:   br i1 %{{[0-9]+}}, label %_llgo_9, label %_llgo_10
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_9:                                          ; preds = %_llgo_8
// CHECK-NEXT:   %{{[0-9]+}} = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{.*}}, i64 13 }, ptr %{{[0-9]+}}, align 8
// CHECK-NEXT:   %{{[0-9]+}} = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %{{[0-9]+}}, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %{{[0-9]+}})
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_10:                                         ; preds = %_llgo_8
// CHECK-NEXT:   %{{[0-9]+}} = alloca %"{{.*}}/runtime/abi.StructField", align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %{{[0-9]+}}, i8 0, i64 56, i1 false)
// CHECK-NEXT:   %{{[0-9]+}} = getelementptr inbounds %"{{.*}}/cl/_testrt/abinamed.eface", ptr %{{[0-9]+}}, i32 0, i32 0
// CHECK-NEXT:   %{{[0-9]+}} = load ptr, ptr %{{[0-9]+}}, align 8
// CHECK-NEXT:   %{{[0-9]+}} = call ptr @"{{.*}}/runtime/abi.(*Type).StructType"(ptr %{{[0-9]+}})
// CHECK-NEXT:   %{{[0-9]+}} = getelementptr inbounds %"{{.*}}/runtime/abi.StructType", ptr %{{[0-9]+}}, i32 0, i32 2
// CHECK-NEXT:   %{{[0-9]+}} = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %{{[0-9]+}}, align 8
// CHECK-NEXT:   %{{[0-9]+}} = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %{{[0-9]+}}, 0
// CHECK-NEXT:   %{{[0-9]+}} = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %{{[0-9]+}}, 1
// CHECK-NEXT:   %{{[0-9]+}} = icmp uge i64 3, %{{[0-9]+}}
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertIndexRange"(i1 %{{[0-9]+}})
// CHECK-NEXT:   %{{[0-9]+}} = getelementptr inbounds %"{{.*}}/runtime/abi.StructField", ptr %{{[0-9]+}}, i64 3
// CHECK-NEXT:   %{{[0-9]+}} = load %"{{.*}}/runtime/abi.StructField", ptr %{{[0-9]+}}, align 8
// CHECK-NEXT:   store %"{{.*}}/runtime/abi.StructField" %{{[0-9]+}}, ptr %{{[0-9]+}}, align 8
// CHECK-NEXT:   %{{[0-9]+}} = getelementptr inbounds %"{{.*}}/runtime/abi.StructField", ptr %{{[0-9]+}}, i32 0, i32 1
// CHECK-NEXT:   %{{[0-9]+}} = load ptr, ptr %{{[0-9]+}}, align 8
// CHECK-NEXT:   %{{[0-9]+}} = call ptr @"{{.*}}/runtime/abi.(*Type).Elem"(ptr %{{[0-9]+}})
// CHECK-NEXT:   %{{[0-9]+}} = getelementptr inbounds %"{{.*}}/cl/_testrt/abinamed.eface", ptr %{{[0-9]+}}, i32 0, i32 0
// CHECK-NEXT:   %{{[0-9]+}} = load ptr, ptr %{{[0-9]+}}, align 8
// CHECK-NEXT:   %{{[0-9]+}} = icmp ne ptr %{{[0-9]+}}, %{{[0-9]+}}
// CHECK-NEXT:   br i1 %{{[0-9]+}}, label %_llgo_11, label %_llgo_12
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_11:                                         ; preds = %_llgo_10
// CHECK-NEXT:   %{{[0-9]+}} = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{.*}}, i64 13 }, ptr %{{[0-9]+}}, align 8
// CHECK-NEXT:   %{{[0-9]+}} = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %{{[0-9]+}}, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %{{[0-9]+}})
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_12:                                         ; preds = %_llgo_10
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

// CHECK-LABEL: define ptr @"{{.*}}/cl/_testrt/abinamed.toEface"(%"{{.*}}/runtime/internal/runtime.eface" %{{[0-9]+}}){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %{{[0-9]+}} = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %{{[0-9]+}}, ptr %{{[0-9]+}}, align 8
// CHECK-NEXT:   ret ptr %{{[0-9]+}}
// CHECK-NEXT: }
func toEface(i any) *eface {
	return (*eface)(unsafe.Pointer(&i))
}
