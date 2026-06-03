// LITTEST
package main

import (
	"unsafe"

	"github.com/goplus/lib/c"
)

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
// CHECK-NEXT:   %20 = call i1 @"{{.*}}/runtime/internal/runtime.StringEqual"(%"{{.*}}/runtime/internal/runtime.String" { ptr @2, i64 3 }, %"{{.*}}/runtime/internal/runtime.String" { ptr @3, i64 3 })
// CHECK-NEXT:   %21 = xor i1 %20, true
// CHECK-NEXT:   br i1 %21, label %_llgo_21, label %_llgo_22
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_21:                                         ; preds = %_llgo_20
// CHECK-NEXT:   %22 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 5 }, ptr %22, align 8
// CHECK-NEXT:   %23 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %22, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %23)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_22:                                         ; preds = %_llgo_20
// CHECK-NEXT:   %24 = load i8, ptr @2, align 1
// CHECK-NEXT:   %25 = icmp ne i8 %24, 97
// CHECK-NEXT:   br i1 %25, label %_llgo_23, label %_llgo_26
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_23:                                         ; preds = %_llgo_25, %_llgo_26, %_llgo_22
// CHECK-NEXT:   %26 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 5 }, ptr %26, align 8
// CHECK-NEXT:   %27 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %26, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %27)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_24:                                         ; preds = %_llgo_25
// CHECK-NEXT:   %28 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %29 = getelementptr inbounds i64, ptr %28, i64 0
// CHECK-NEXT:   %30 = getelementptr inbounds i64, ptr %28, i64 1
// CHECK-NEXT:   store i64 1, ptr %29, align 8
// CHECK-NEXT:   store i64 2, ptr %30, align 8
// CHECK-NEXT:   %31 = getelementptr inbounds i64, ptr %28, i64 0
// CHECK-NEXT:   %32 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %31, 0
// CHECK-NEXT:   %33 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %32, i64 2, 1
// CHECK-NEXT:   %34 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %33, i64 2, 2
// CHECK-NEXT:   %35 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %34, 0
// CHECK-NEXT:   %36 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %34, 1
// CHECK-NEXT:   %37 = icmp uge i64 0, %36
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertIndexRange"(i1 %37)
// CHECK-NEXT:   %38 = getelementptr inbounds i64, ptr %35, i64 0
// CHECK-NEXT:   %39 = load i64, ptr %38, align 8
// CHECK-NEXT:   %40 = icmp ne i64 %39, 1
// CHECK-NEXT:   br i1 %40, label %_llgo_27, label %_llgo_29
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_25:                                         ; preds = %_llgo_26
// CHECK-NEXT:   %41 = load i8, ptr getelementptr inbounds (i8, ptr @2, i64 2), align 1
// CHECK-NEXT:   %42 = icmp ne i8 %41, 99
// CHECK-NEXT:   br i1 %42, label %_llgo_23, label %_llgo_24
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_26:                                         ; preds = %_llgo_22
// CHECK-NEXT:   %43 = load i8, ptr getelementptr inbounds (i8, ptr @2, i64 1), align 1
// CHECK-NEXT:   %44 = icmp ne i8 %43, 98
// CHECK-NEXT:   br i1 %44, label %_llgo_23, label %_llgo_25
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_27:                                         ; preds = %_llgo_29, %_llgo_24
// CHECK-NEXT:   %45 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 5 }, ptr %45, align 8
// CHECK-NEXT:   %46 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %45, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %46)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_28:                                         ; preds = %_llgo_29
// CHECK-NEXT:   %47 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %34, 0
// CHECK-NEXT:   %48 = load i64, ptr %47, align 8
// CHECK-NEXT:   %49 = icmp ne i64 %48, 1
// CHECK-NEXT:   br i1 %49, label %_llgo_30, label %_llgo_31
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_29:                                         ; preds = %_llgo_24
// CHECK-NEXT:   %50 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %34, 0
// CHECK-NEXT:   %51 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %34, 1
// CHECK-NEXT:   %52 = icmp uge i64 1, %51
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertIndexRange"(i1 %52)
// CHECK-NEXT:   %53 = getelementptr inbounds i64, ptr %50, i64 1
// CHECK-NEXT:   %54 = load i64, ptr %53, align 8
// CHECK-NEXT:   %55 = icmp ne i64 %54, 2
// CHECK-NEXT:   br i1 %55, label %_llgo_27, label %_llgo_28
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_30:                                         ; preds = %_llgo_28
// CHECK-NEXT:   %56 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 5 }, ptr %56, align 8
// CHECK-NEXT:   %57 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %56, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %57)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_31:                                         ; preds = %_llgo_28
// CHECK-NEXT:   %58 = icmp ne i64 ptrtoint (ptr getelementptr (i8, ptr null, i64 1) to i64), 1
// CHECK-NEXT:   br i1 %58, label %_llgo_32, label %_llgo_33
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_32:                                         ; preds = %_llgo_31
// CHECK-NEXT:   %59 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 5 }, ptr %59, align 8
// CHECK-NEXT:   %60 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %59, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %60)
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
