// LITTEST
package main

import (
	"unsafe"

	"github.com/goplus/llgo/runtime/abi"
)

// CHECK-LINE: @0 = private unnamed_addr constant [5 x i8] c"int32", align 1
// CHECK-LINE: @1 = private unnamed_addr constant [14 x i8] c"abi rune error", align 1
// CHECK-LINE: @3 = private unnamed_addr constant [5 x i8] c"uint8", align 1
// CHECK-LINE: @4 = private unnamed_addr constant [14 x i8] c"abi byte error", align 1

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/abitype.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testrt/abitype.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testrt/abitype.init$guard", align 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/abi.init"()
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

type eface struct {
	typ  *abi.Type
	data unsafe.Pointer
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/abitype.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store i32 0, ptr %1, align 4
// CHECK-NEXT:   %2 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int32, ptr undef }, ptr %1, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %2, ptr %0, align 8
// CHECK-NEXT:   %3 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %3)
// CHECK-NEXT:   %4 = getelementptr inbounds %"{{.*}}/cl/_testrt/abitype.eface", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %5 = load ptr, ptr %4, align 8
// CHECK-NEXT:   %6 = call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/runtime/abi.(*Type).String"(ptr %5)
// CHECK-NEXT:   %7 = call i1 @"{{.*}}/runtime/internal/runtime.StringEqual"(%"{{.*}}/runtime/internal/runtime.String" %6, %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 5 })
// CHECK-NEXT:   %8 = xor i1 %7, true
// CHECK-NEXT:   br i1 %8, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %9 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @1, i64 14 }, ptr %9, align 8
// CHECK-NEXT:   %10 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %9, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %10)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %11 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// CHECK-NEXT:   store i8 0, ptr %11, align 1
// CHECK-NEXT:   %12 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint8, ptr undef }, ptr %11, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %12, ptr %0, align 8
// CHECK-NEXT:   %13 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %13)
// CHECK-NEXT:   %14 = getelementptr inbounds %"{{.*}}/cl/_testrt/abitype.eface", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %15 = load ptr, ptr %14, align 8
// CHECK-NEXT:   %16 = call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/runtime/abi.(*Type).String"(ptr %15)
// CHECK-NEXT:   %17 = call i1 @"{{.*}}/runtime/internal/runtime.StringEqual"(%"{{.*}}/runtime/internal/runtime.String" %16, %"{{.*}}/runtime/internal/runtime.String" { ptr @3, i64 5 })
// CHECK-NEXT:   %18 = xor i1 %17, true
// CHECK-NEXT:   br i1 %18, label %_llgo_3, label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %19 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @4, i64 14 }, ptr %19, align 8
// CHECK-NEXT:   %20 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %19, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %20)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_2
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

func main() {
	var v any = rune(0)
	t := (*eface)(unsafe.Pointer(&v)).typ
	if t.String() != "int32" {
		panic("abi rune error")
	}
	v = byte(0)
	t = (*eface)(unsafe.Pointer(&v)).typ
	if t.String() != "uint8" {
		panic("abi byte error")
	}
}

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequal8"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.memequal8"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }
