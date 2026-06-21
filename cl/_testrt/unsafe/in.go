// LITTEST
package main

import (
	"unsafe"

	"github.com/goplus/lib/c"
)

// CHECK: {{^}}@0 = private unnamed_addr constant [5 x i8] c"error", align 1{{$}}
// CHECK: {{^}}@2 = private unnamed_addr constant [4 x i8] c"abc\00", align 1{{$}}
// CHECK: {{^}}@3 = private unnamed_addr constant [31 x i8] c"unsafe.String: len out of range", align 1{{$}}
// CHECK: {{^}}@4 = private unnamed_addr constant [47 x i8] c"unsafe.String: nil pointer with non-zero length", align 1{{$}}
// CHECK: {{^}}@5 = private unnamed_addr constant [3 x i8] c"abc", align 1{{$}}
// CHECK: {{^}}@6 = private unnamed_addr constant [30 x i8] c"unsafe.Slice: len out of range", align 1{{$}}
// CHECK: {{^}}@7 = private unnamed_addr constant [46 x i8] c"unsafe.Slice: nil pointer with non-zero length", align 1{{$}}

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/unsafe.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testrt/unsafe.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testrt/unsafe.init$guard", align 1
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

//llgo:type C
type T func()

type M struct {
	fn T
	v  int
}

type N struct {
	fn func()
	v  int
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/unsafe.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   br i1 false, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 5 }, ptr %0, align 8
// CHECK-NEXT:   %1 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %0, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %1)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   br i1 false, label %_llgo_3, label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %2 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 5 }, ptr %2, align 8
// CHECK-NEXT:   %3 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %2, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %3)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_2
// CHECK-NEXT:   br i1 false, label %_llgo_5, label %_llgo_6
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_4
// CHECK-NEXT:   %4 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 5 }, ptr %4, align 8
// CHECK-NEXT:   %5 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %4, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %5)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_4
// CHECK-NEXT:   br i1 false, label %_llgo_7, label %_llgo_8
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_7:                                          ; preds = %_llgo_6
// CHECK-NEXT:   %6 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 5 }, ptr %6, align 8
// CHECK-NEXT:   %7 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %6, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %7)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_8:                                          ; preds = %_llgo_6
// CHECK-NEXT:   br i1 false, label %_llgo_9, label %_llgo_10
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_9:                                          ; preds = %_llgo_8
// CHECK-NEXT:   %8 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 5 }, ptr %8, align 8
// CHECK-NEXT:   %9 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %8, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %9)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_10:                                         ; preds = %_llgo_8
// CHECK-NEXT:   br i1 false, label %_llgo_11, label %_llgo_12
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_11:                                         ; preds = %_llgo_10
// CHECK-NEXT:   %10 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 5 }, ptr %10, align 8
// CHECK-NEXT:   %11 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %10, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %11)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_12:                                         ; preds = %_llgo_10
// CHECK-NEXT:   br i1 false, label %_llgo_13, label %_llgo_14
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_13:                                         ; preds = %_llgo_12
// CHECK-NEXT:   %12 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 5 }, ptr %12, align 8
// CHECK-NEXT:   %13 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %12, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %13)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_14:                                         ; preds = %_llgo_12
// CHECK-NEXT:   br i1 false, label %_llgo_15, label %_llgo_16
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_15:                                         ; preds = %_llgo_14
// CHECK-NEXT:   %14 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 5 }, ptr %14, align 8
// CHECK-NEXT:   %15 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %14, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %15)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_16:                                         ; preds = %_llgo_14
// CHECK-NEXT:   br i1 false, label %_llgo_17, label %_llgo_18
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_17:                                         ; preds = %_llgo_16
// CHECK-NEXT:   %16 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 5 }, ptr %16, align 8
// CHECK-NEXT:   %17 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %16, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %17)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_18:                                         ; preds = %_llgo_16
// CHECK-NEXT:   br i1 false, label %_llgo_19, label %_llgo_20
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_19:                                         ; preds = %_llgo_18
// CHECK-NEXT:   %18 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 5 }, ptr %18, align 8
// CHECK-NEXT:   %19 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %18, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %19)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_20:                                         ; preds = %_llgo_18
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertRuntimeError"(i1 false, %"{{.*}}/runtime/internal/runtime.String" { ptr @3, i64 31 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertRuntimeError"(i1 false, %"{{.*}}/runtime/internal/runtime.String" { ptr @4, i64 47 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertRuntimeError"(i1 false, %"{{.*}}/runtime/internal/runtime.String" { ptr @3, i64 31 })
// CHECK-NEXT:   %20 = icmp ult i64 add (i64 ptrtoint (ptr @2 to i64), i64 2), ptrtoint (ptr @2 to i64)
// CHECK-NEXT:   %21 = and i1 true, %20
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertRuntimeError"(i1 %21, %"{{.*}}/runtime/internal/runtime.String" { ptr @3, i64 31 })
// CHECK-NEXT:   %22 = call i1 @"{{.*}}/runtime/internal/runtime.StringEqual"(%"{{.*}}/runtime/internal/runtime.String" { ptr @2, i64 3 }, %"{{.*}}/runtime/internal/runtime.String" { ptr @5, i64 3 })
// CHECK-NEXT:   %23 = xor i1 %22, true
// CHECK-NEXT:   br i1 %23, label %_llgo_21, label %_llgo_22
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_21:                                         ; preds = %_llgo_20
// CHECK-NEXT:   %24 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 5 }, ptr %24, align 8
// CHECK-NEXT:   %25 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %24, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %25)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_22:                                         ; preds = %_llgo_20
// CHECK-NEXT:   %26 = load i8, ptr @2, align 1
// CHECK-NEXT:   %27 = icmp ne i8 %26, 97
// CHECK-NEXT:   br i1 %27, label %_llgo_23, label %_llgo_26
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_23:                                         ; preds = %_llgo_25, %_llgo_26, %_llgo_22
// CHECK-NEXT:   %28 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 5 }, ptr %28, align 8
// CHECK-NEXT:   %29 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %28, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %29)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_24:                                         ; preds = %_llgo_25
// CHECK-NEXT:   %30 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %31 = getelementptr inbounds i64, ptr %30, i64 0
// CHECK-NEXT:   %32 = getelementptr inbounds i64, ptr %30, i64 1
// CHECK-NEXT:   store i64 1, ptr %31, align 8
// CHECK-NEXT:   store i64 2, ptr %32, align 8
// CHECK-NEXT:   %33 = getelementptr inbounds i64, ptr %30, i64 0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertRuntimeError"(i1 false, %"{{.*}}/runtime/internal/runtime.String" { ptr @6, i64 30 })
// CHECK-NEXT:   %34 = icmp eq ptr %33, null
// CHECK-NEXT:   %35 = and i1 %34, true
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertRuntimeError"(i1 %35, %"{{.*}}/runtime/internal/runtime.String" { ptr @7, i64 46 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertRuntimeError"(i1 false, %"{{.*}}/runtime/internal/runtime.String" { ptr @6, i64 30 })
// CHECK-NEXT:   %36 = ptrtoint ptr %33 to i64
// CHECK-NEXT:   %37 = add i64 %36, 15
// CHECK-NEXT:   %38 = icmp ult i64 %37, %36
// CHECK-NEXT:   %39 = and i1 true, %38
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertRuntimeError"(i1 %39, %"{{.*}}/runtime/internal/runtime.String" { ptr @6, i64 30 })
// CHECK-NEXT:   %40 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %33, 0
// CHECK-NEXT:   %41 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %40, i64 2, 1
// CHECK-NEXT:   %42 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %41, i64 2, 2
// CHECK-NEXT:   %43 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %42, 0
// CHECK-NEXT:   %44 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %42, 1
// CHECK-NEXT:   %45 = icmp uge i64 0, %44
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.CheckIndexRange"(i1 %45, i64 0, i1 true, i64 %44)
// CHECK-NEXT:   %46 = getelementptr inbounds i64, ptr %43, i64 0
// CHECK-NEXT:   %47 = load i64, ptr %46, align 8
// CHECK-NEXT:   %48 = icmp ne i64 %47, 1
// CHECK-NEXT:   br i1 %48, label %_llgo_27, label %_llgo_29
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_25:                                         ; preds = %_llgo_26
// CHECK-NEXT:   %49 = load i8, ptr getelementptr inbounds (i8, ptr @2, i64 2), align 1
// CHECK-NEXT:   %50 = icmp ne i8 %49, 99
// CHECK-NEXT:   br i1 %50, label %_llgo_23, label %_llgo_24
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_26:                                         ; preds = %_llgo_22
// CHECK-NEXT:   %51 = load i8, ptr getelementptr inbounds (i8, ptr @2, i64 1), align 1
// CHECK-NEXT:   %52 = icmp ne i8 %51, 98
// CHECK-NEXT:   br i1 %52, label %_llgo_23, label %_llgo_25
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_27:                                         ; preds = %_llgo_29, %_llgo_24
// CHECK-NEXT:   %53 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 5 }, ptr %53, align 8
// CHECK-NEXT:   %54 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %53, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %54)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_28:                                         ; preds = %_llgo_29
// CHECK-NEXT:   %55 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %42, 0
// CHECK-NEXT:   %56 = load i64, ptr %55, align 8
// CHECK-NEXT:   %57 = icmp ne i64 %56, 1
// CHECK-NEXT:   br i1 %57, label %_llgo_30, label %_llgo_31
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_29:                                         ; preds = %_llgo_24
// CHECK-NEXT:   %58 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %42, 0
// CHECK-NEXT:   %59 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %42, 1
// CHECK-NEXT:   %60 = icmp uge i64 1, %59
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.CheckIndexRange"(i1 %60, i64 1, i1 true, i64 %59)
// CHECK-NEXT:   %61 = getelementptr inbounds i64, ptr %58, i64 1
// CHECK-NEXT:   %62 = load i64, ptr %61, align 8
// CHECK-NEXT:   %63 = icmp ne i64 %62, 2
// CHECK-NEXT:   br i1 %63, label %_llgo_27, label %_llgo_28
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_30:                                         ; preds = %_llgo_28
// CHECK-NEXT:   %64 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 5 }, ptr %64, align 8
// CHECK-NEXT:   %65 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %64, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %65)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_31:                                         ; preds = %_llgo_28
// CHECK-NEXT:   %66 = icmp ne i64 ptrtoint (ptr getelementptr (i8, ptr null, i64 1) to i64), 1
// CHECK-NEXT:   br i1 %66, label %_llgo_32, label %_llgo_33
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_32:                                         ; preds = %_llgo_31
// CHECK-NEXT:   %67 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 5 }, ptr %67, align 8
// CHECK-NEXT:   %68 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %67, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %68)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_33:                                         ; preds = %_llgo_31
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

func main() {
	if unsafe.Sizeof(*(*T)(nil)) != unsafe.Sizeof(0) {
		panic("error")
	}
	if unsafe.Sizeof(*(*M)(nil)) != unsafe.Sizeof([2]int{}) {
		panic("error")
	}
	// TODO(lijie): inconsistent with golang
	if unsafe.Sizeof(*(*N)(nil)) != unsafe.Sizeof([3]int{}) {
		panic("error")
	}

	if unsafe.Alignof(*(*T)(nil)) != unsafe.Alignof(0) {
		panic("error")
	}
	if unsafe.Alignof(*(*M)(nil)) != unsafe.Alignof([2]int{}) {
		panic("error")
	}
	if unsafe.Alignof(*(*N)(nil)) != unsafe.Alignof([3]int{}) {
		panic("error")
	}

	if unsafe.Offsetof(M{}.fn) != 0 {
		panic("error")
	}
	if unsafe.Offsetof(M{}.v) != unsafe.Sizeof(int(0)) {
		panic("error")
	}
	if unsafe.Offsetof(N{}.fn) != 0 {
		panic("error")
	}
	// TODO(lijie): inconsistent with golang
	if unsafe.Offsetof(N{}.v) != unsafe.Sizeof([2]int{}) {
		panic("error")
	}

	s := unsafe.String((*byte)(unsafe.Pointer(c.Str("abc"))), 3)
	if s != "abc" {
		panic("error")
	}

	p := unsafe.StringData(s)
	arr := (*[3]byte)(unsafe.Pointer(p))
	if arr[0] != 'a' || arr[1] != 'b' || arr[2] != 'c' {
		panic("error")
	}

	intArr := [2]int{1, 2}
	pi := &intArr[0]
	intSlice := unsafe.Slice(pi, 2)
	if intSlice[0] != 1 || intSlice[1] != 2 {
		panic("error")
	}

	pi = unsafe.SliceData(intSlice)
	if *pi != 1 {
		panic("error")
	}

	if uintptr(unsafe.Add(unsafe.Pointer(nil), 1)) != 1 {
		panic("error")
	}

}
