// LITTEST
package main

import (
	"unsafe"

	"github.com/goplus/llgo/runtime/abi"
)

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store i32 0, ptr %1, align 4
// CHECK-NEXT:   %2 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int32, ptr undef }, ptr %1, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %2, ptr %0, align 8
// CHECK-NEXT:   %3 = getelementptr inbounds %main.eface, ptr %0, i32 0, i32 0
// CHECK-NEXT:   %4 = load ptr, ptr %3, align 8
// CHECK-NEXT:   %5 = call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/runtime/abi.(*Type).String"(ptr %4)
// CHECK-NEXT:   %6 = call i1 @"{{.*}}/runtime/internal/runtime.StringEqual"(%"{{.*}}/runtime/internal/runtime.String" %5, %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 5 })
// CHECK-NEXT:   %7 = xor i1 %6, true
// CHECK-NEXT:   br i1 %7, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %8 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @1, i64 14 }, ptr %8, align 8
// CHECK-NEXT:   %9 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %8, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %9)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %10 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// CHECK-NEXT:   store i8 0, ptr %10, align 1
// CHECK-NEXT:   %11 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint8, ptr undef }, ptr %10, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %11, ptr %0, align 8
// CHECK-NEXT:   %12 = getelementptr inbounds %main.eface, ptr %0, i32 0, i32 0
// CHECK-NEXT:   %13 = load ptr, ptr %12, align 8
// CHECK-NEXT:   %14 = call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/runtime/abi.(*Type).String"(ptr %13)
// CHECK-NEXT:   %15 = call i1 @"{{.*}}/runtime/internal/runtime.StringEqual"(%"{{.*}}/runtime/internal/runtime.String" %14, %"{{.*}}/runtime/internal/runtime.String" { ptr @3, i64 5 })
// CHECK-NEXT:   %16 = xor i1 %15, true
// CHECK-NEXT:   br i1 %16, label %_llgo_3, label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %17 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @4, i64 14 }, ptr %17, align 8
// CHECK-NEXT:   %18 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %17, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %18)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_2
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// ESCAPE-LABEL: define void @main.main(){{.*}} {
// ESCAPE-NEXT: _llgo_0:
// ESCAPE-NEXT:   %.stack = alloca i8, i64 16, align 8
// ESCAPE-NEXT:   call void @llvm.memset.p0.i64(ptr %.stack, i8 0, i64 16, i1 false)
// ESCAPE-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// ESCAPE-NEXT:   store i32 0, ptr %0, align 4
// ESCAPE-NEXT:   %1 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int32, ptr undef }, ptr %0, 1
// ESCAPE-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %1, ptr %.stack, align 8
// ESCAPE-NEXT:   %2 = getelementptr inbounds %main.eface, ptr %.stack, i32 0, i32 0
// ESCAPE-NEXT:   %3 = load ptr, ptr %2, align 8
// ESCAPE-NEXT:   %4 = call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/runtime/abi.(*Type).String"(ptr %3)
// ESCAPE-NEXT:   %5 = call i1 @"{{.*}}/runtime/internal/runtime.StringEqual"(%"{{.*}}/runtime/internal/runtime.String" %4, %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 5 })
// ESCAPE-NEXT:   %6 = xor i1 %5, true
// ESCAPE-NEXT:   br i1 %6, label %_llgo_1, label %_llgo_2
// ESCAPE-EMPTY:
// ESCAPE-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// ESCAPE-NEXT:   %7 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// ESCAPE-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @1, i64 14 }, ptr %7, align 8
// ESCAPE-NEXT:   %8 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %7, 1
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %8)
// ESCAPE-NEXT:   unreachable
// ESCAPE-EMPTY:
// ESCAPE-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// ESCAPE-NEXT:   %9 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// ESCAPE-NEXT:   store i8 0, ptr %9, align 1
// ESCAPE-NEXT:   %10 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint8, ptr undef }, ptr %9, 1
// ESCAPE-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %10, ptr %.stack, align 8
// ESCAPE-NEXT:   %11 = getelementptr inbounds %main.eface, ptr %.stack, i32 0, i32 0
// ESCAPE-NEXT:   %12 = load ptr, ptr %11, align 8
// ESCAPE-NEXT:   %13 = call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/runtime/abi.(*Type).String"(ptr %12)
// ESCAPE-NEXT:   %14 = call i1 @"{{.*}}/runtime/internal/runtime.StringEqual"(%"{{.*}}/runtime/internal/runtime.String" %13, %"{{.*}}/runtime/internal/runtime.String" { ptr @3, i64 5 })
// ESCAPE-NEXT:   %15 = xor i1 %14, true
// ESCAPE-NEXT:   br i1 %15, label %_llgo_3, label %_llgo_4
// ESCAPE-EMPTY:
// ESCAPE-NEXT: _llgo_3:                                          ; preds = %_llgo_2
// ESCAPE-NEXT:   %16 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// ESCAPE-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @4, i64 14 }, ptr %16, align 8
// ESCAPE-NEXT:   %17 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %16, 1
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %17)
// ESCAPE-NEXT:   unreachable
// ESCAPE-EMPTY:
// ESCAPE-NEXT: _llgo_4:                                          ; preds = %_llgo_2
// ESCAPE-NEXT:   ret void
// ESCAPE-NEXT: }

type eface struct {
	typ  *abi.Type
	data unsafe.Pointer
}

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
