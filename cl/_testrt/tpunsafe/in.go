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

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 12)
// CHECK-NEXT:   call void @"main.(*M[bool]).check"(ptr %0, i64 1, i64 8, i64 1)
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 32)
// CHECK-NEXT:   call void @"main.(*M[int64]).check"(ptr %1, i64 8, i64 16, i64 8)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func main() {
	m1 := M[bool]{}
	m1.check(1, 8, 1)
	m2 := M[int64]{}
	m2.check(8, 16, 8)
}

// CHECK-LABEL: define linkonce void @"main.(*M[bool]).check"(ptr %0, i64 %1, i64 %2, i64 %3){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %4 = getelementptr inbounds %"main.M[bool]", ptr %0, i32 0, i32 2
// CHECK-NEXT:   %5 = icmp eq ptr %0, null
// CHECK-NEXT:   br i1 %5, label %17, label %18
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %18
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 4 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 1)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @1, i64 4 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintUint"(i64 %1)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %6 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @2, i64 20 }, ptr %6, align 8
// CHECK-NEXT:   %7 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %6, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %7)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %18
// CHECK-NEXT:   %8 = getelementptr inbounds %"main.M[bool]", ptr %0, i32 0, i32 2
// CHECK-NEXT:   %9 = icmp eq ptr %0, null
// CHECK-NEXT:   br i1 %9, label %21, label %22
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %22
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 4 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 8)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @1, i64 4 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintUint"(i64 %2)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %10 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @4, i64 21 }, ptr %10, align 8
// CHECK-NEXT:   %11 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %10, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %11)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %22
// CHECK-NEXT:   %12 = getelementptr inbounds %"main.M[bool]", ptr %0, i32 0, i32 2
// CHECK-NEXT:   %13 = getelementptr inbounds %"main.N[bool]", ptr %12, i32 0, i32 1
// CHECK-NEXT:   %14 = icmp eq ptr %0, null
// CHECK-NEXT:   br i1 %14, label %25, label %26
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %29
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 4 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 1)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @1, i64 4 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintUint"(i64 %3)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %15 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @4, i64 21 }, ptr %15, align 8
// CHECK-NEXT:   %16 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %15, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %16)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %29
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: 17:                                               ; preds = %_llgo_0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 18:                                               ; preds = %_llgo_0
// CHECK-NEXT:   %19 = load %"main.N[bool]", ptr %4, align 1
// CHECK-NEXT:   %20 = icmp ne i64 1, %1
// CHECK-NEXT:   br i1 %20, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: 21:                                               ; preds = %_llgo_2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 22:                                               ; preds = %_llgo_2
// CHECK-NEXT:   %23 = load %"main.N[bool]", ptr %8, align 1
// CHECK-NEXT:   %24 = icmp ne i64 8, %2
// CHECK-NEXT:   br i1 %24, label %_llgo_3, label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: 25:                                               ; preds = %_llgo_4
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 26:                                               ; preds = %_llgo_4
// CHECK-NEXT:   %27 = icmp eq ptr %12, null
// CHECK-NEXT:   br i1 %27, label %28, label %29
// CHECK-EMPTY:
// CHECK-NEXT: 28:                                               ; preds = %26
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 29:                                               ; preds = %26
// CHECK-NEXT:   %30 = load i1, ptr %13, align 1
// CHECK-NEXT:   %31 = icmp ne i64 1, %3
// CHECK-NEXT:   br i1 %31, label %_llgo_5, label %_llgo_6
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

// CHECK-LABEL: define linkonce void @"main.(*M[int64]).check"(ptr %0, i64 %1, i64 %2, i64 %3){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %4 = getelementptr inbounds %"main.M[int64]", ptr %0, i32 0, i32 2
// CHECK-NEXT:   %5 = icmp eq ptr %0, null
// CHECK-NEXT:   br i1 %5, label %17, label %18
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %18
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 4 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 8)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @1, i64 4 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintUint"(i64 %1)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %6 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @2, i64 20 }, ptr %6, align 8
// CHECK-NEXT:   %7 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %6, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %7)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %18
// CHECK-NEXT:   %8 = getelementptr inbounds %"main.M[int64]", ptr %0, i32 0, i32 2
// CHECK-NEXT:   %9 = icmp eq ptr %0, null
// CHECK-NEXT:   br i1 %9, label %21, label %22
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %22
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 4 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 16)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @1, i64 4 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintUint"(i64 %2)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %10 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @4, i64 21 }, ptr %10, align 8
// CHECK-NEXT:   %11 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %10, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %11)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %22
// CHECK-NEXT:   %12 = getelementptr inbounds %"main.M[int64]", ptr %0, i32 0, i32 2
// CHECK-NEXT:   %13 = getelementptr inbounds %"main.N[int64]", ptr %12, i32 0, i32 1
// CHECK-NEXT:   %14 = icmp eq ptr %0, null
// CHECK-NEXT:   br i1 %14, label %25, label %26
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %29
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 4 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 8)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @1, i64 4 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintUint"(i64 %3)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %15 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @4, i64 21 }, ptr %15, align 8
// CHECK-NEXT:   %16 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %15, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %16)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %29
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: 17:                                               ; preds = %_llgo_0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 18:                                               ; preds = %_llgo_0
// CHECK-NEXT:   %19 = load %"main.N[int64]", ptr %4, align 8
// CHECK-NEXT:   %20 = icmp ne i64 8, %1
// CHECK-NEXT:   br i1 %20, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: 21:                                               ; preds = %_llgo_2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 22:                                               ; preds = %_llgo_2
// CHECK-NEXT:   %23 = load %"main.N[int64]", ptr %8, align 8
// CHECK-NEXT:   %24 = icmp ne i64 16, %2
// CHECK-NEXT:   br i1 %24, label %_llgo_3, label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: 25:                                               ; preds = %_llgo_4
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 26:                                               ; preds = %_llgo_4
// CHECK-NEXT:   %27 = icmp eq ptr %12, null
// CHECK-NEXT:   br i1 %27, label %28, label %29
// CHECK-EMPTY:
// CHECK-NEXT: 28:                                               ; preds = %26
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 29:                                               ; preds = %26
// CHECK-NEXT:   %30 = load i64, ptr %13, align 8
// CHECK-NEXT:   %31 = icmp ne i64 8, %3
// CHECK-NEXT:   br i1 %31, label %_llgo_5, label %_llgo_6
// CHECK-NEXT: }
