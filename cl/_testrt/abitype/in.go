// LITTEST
package main

import (
	"unsafe"

	"github.com/goplus/llgo/runtime/abi"
)

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/abitype.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store i32 0, ptr %1, align 4
// CHECK-NEXT:   %2 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int32, ptr undef }, ptr %1, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %2, ptr %0, align 8
// CHECK-NEXT:   %3 = getelementptr inbounds %"{{.*}}/cl/_testrt/abitype.eface", ptr %0, i32 0, i32 0
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
// CHECK-NEXT:   %12 = getelementptr inbounds %"{{.*}}/cl/_testrt/abitype.eface", ptr %0, i32 0, i32 0
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
