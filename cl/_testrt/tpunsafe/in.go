// LITTEST
package main

import (
	"unsafe"
)

type N[T any] struct {
	n1 T
	n2 T
}

type M[T any] struct {
	m0 T
	m1 int32
	m2 N[T]
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/tpunsafe.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 12)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/tpunsafe.(*M[bool]).check"(ptr %0, i64 1, i64 8, i64 1)
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 32)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/tpunsafe.(*M[int64]).check"(ptr %1, i64 8, i64 16, i64 8)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func main() {
	m1 := M[bool]{}
	m1.check(1, 8, 1)
	m2 := M[int64]{}
	m2.check(8, 16, 8)
}

// CHECK-LABEL: define linkonce void @"{{.*}}/cl/_testrt/tpunsafe.(*M[bool]).check"(ptr %0, i64 %1, i64 %2, i64 %3){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %4 = getelementptr inbounds %"{{.*}}/cl/_testrt/tpunsafe.M[bool]", ptr %0, i32 0, i32 2
// CHECK-NEXT:   %5 = load %"{{.*}}/cl/_testrt/tpunsafe.N[bool]", ptr %4, align 1
// CHECK-NEXT:   %6 = icmp ne i64 1, %1
// CHECK-NEXT:   br i1 %6, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 4 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 1)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @1, i64 4 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintUint"(i64 %1)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %7 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @2, i64 20 }, ptr %7, align 8
// CHECK-NEXT:   %8 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %7, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %8)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %9 = getelementptr inbounds %"{{.*}}/cl/_testrt/tpunsafe.M[bool]", ptr %0, i32 0, i32 2
// CHECK-NEXT:   %10 = load %"{{.*}}/cl/_testrt/tpunsafe.N[bool]", ptr %9, align 1
// CHECK-NEXT:   %11 = icmp ne i64 8, %2
// CHECK-NEXT:   br i1 %11, label %_llgo_3, label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 4 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 8)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @1, i64 4 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintUint"(i64 %2)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %12 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @4, i64 21 }, ptr %12, align 8
// CHECK-NEXT:   %13 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %12, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %13)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %14 = getelementptr inbounds %"{{.*}}/cl/_testrt/tpunsafe.M[bool]", ptr %0, i32 0, i32 2
// CHECK-NEXT:   %15 = getelementptr inbounds %"{{.*}}/cl/_testrt/tpunsafe.N[bool]", ptr %14, i32 0, i32 1
// CHECK-NEXT:   %16 = load i1, ptr %15, align 1
// CHECK-NEXT:   %17 = icmp ne i64 1, %3
// CHECK-NEXT:   br i1 %17, label %_llgo_5, label %_llgo_6
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_4
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 4 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 1)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @1, i64 4 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintUint"(i64 %3)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %18 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @4, i64 21 }, ptr %18, align 8
// CHECK-NEXT:   %19 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %18, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %19)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_4
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func (m *M[T]) check(align, offset1, offset2 uintptr) {
	if v := unsafe.Alignof(m.m2); v != align {
		println("have", v, "want", align)
		panic("unsafe.Alignof error")
	}
	if v := unsafe.Offsetof(m.m2); v != offset1 {
		println("have", v, "want", offset1)
		panic("unsafe.Offsetof error")
	}
	if v := unsafe.Offsetof(m.m2.n2); v != offset2 {
		println("have", v, "want", offset2)
		panic("unsafe.Offsetof error")
	}
}

// CHECK-LABEL: define linkonce void @"{{.*}}/cl/_testrt/tpunsafe.(*M[int64]).check"(ptr %0, i64 %1, i64 %2, i64 %3){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %4 = getelementptr inbounds %"{{.*}}/cl/_testrt/tpunsafe.M[int64]", ptr %0, i32 0, i32 2
// CHECK-NEXT:   %5 = load %"{{.*}}/cl/_testrt/tpunsafe.N[int64]", ptr %4, align 8
// CHECK-NEXT:   %6 = icmp ne i64 8, %1
// CHECK-NEXT:   br i1 %6, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 4 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 8)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @1, i64 4 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintUint"(i64 %1)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %7 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @2, i64 20 }, ptr %7, align 8
// CHECK-NEXT:   %8 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %7, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %8)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %9 = getelementptr inbounds %"{{.*}}/cl/_testrt/tpunsafe.M[int64]", ptr %0, i32 0, i32 2
// CHECK-NEXT:   %10 = load %"{{.*}}/cl/_testrt/tpunsafe.N[int64]", ptr %9, align 8
// CHECK-NEXT:   %11 = icmp ne i64 16, %2
// CHECK-NEXT:   br i1 %11, label %_llgo_3, label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 4 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 16)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @1, i64 4 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintUint"(i64 %2)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %12 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @4, i64 21 }, ptr %12, align 8
// CHECK-NEXT:   %13 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %12, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %13)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %14 = getelementptr inbounds %"{{.*}}/cl/_testrt/tpunsafe.M[int64]", ptr %0, i32 0, i32 2
// CHECK-NEXT:   %15 = getelementptr inbounds %"{{.*}}/cl/_testrt/tpunsafe.N[int64]", ptr %14, i32 0, i32 1
// CHECK-NEXT:   %16 = load i64, ptr %15, align 8
// CHECK-NEXT:   %17 = icmp ne i64 8, %3
// CHECK-NEXT:   br i1 %17, label %_llgo_5, label %_llgo_6
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_4
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 4 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 8)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @1, i64 4 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintUint"(i64 %3)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %18 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @4, i64 21 }, ptr %18, align 8
// CHECK-NEXT:   %19 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %18, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %19)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_4
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
