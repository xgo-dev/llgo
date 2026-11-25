// LITTEST
// Scope: common
package main

import (
	"unsafe"

	q "github.com/xgo-dev/llgo/cmd/cl/_testrt/qsortfn/qsort"
)

//llgo:cdecl
type Comp func(a, b unsafe.Pointer) int32

//go:linkname cstr llgo.cstr
func cstr(string) *int8

//go:linkname printf C.printf
func printf(format *int8, __llgo_va_list ...any) int32

//go:linkname qsortLocal C.qsort
func qsortLocal(base unsafe.Pointer, count, elem uintptr, compar Comp)

//go:linkname qsortUnnamed C.qsort
func qsortUnnamed(base unsafe.Pointer, count, elem uintptr, compar func(a, b unsafe.Pointer) int32)

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: call void @main.sortLocalNamed()
// CHECK: call void @main.sortLocalToUnnamed()
// CHECK: call void @main.sortUnnamedToImported()
// CHECK: call void @main.sortExplicitNamedConversion()

// CHECK-LABEL: define void @main.sortExplicitNamedConversion(){{.*}} {
// CHECK: call void @qsort({{.*}}ptr @"main.sortExplicitNamedConversion$1")
// CHECK-LABEL: define i32 @"main.sortExplicitNamedConversion$1"(ptr %0, ptr %1){{.*}} {
// CHECK: sub i64
// CHECK: trunc i64 {{%[0-9]+}} to i32

// CHECK-LABEL: define void @main.sortLocalNamed(){{.*}} {
// CHECK: call void @qsort({{.*}}ptr @"main.sortLocalNamed$1")
// CHECK-LABEL: define i32 @"main.sortLocalNamed$1"(ptr %0, ptr %1){{.*}} {
// CHECK: [[LOCAL_A:%[0-9]+]] = load i64, ptr %0
// CHECK-NEXT: [[LOCAL_B:%[0-9]+]] = load i64, ptr %1
// CHECK-NEXT: [[LOCAL_DIFF:%[0-9]+]] = sub i64 [[LOCAL_A]], [[LOCAL_B]]
// CHECK-NEXT: [[LOCAL_RESULT:%[0-9]+]] = trunc i64 [[LOCAL_DIFF]] to i32
// CHECK-NEXT: ret i32 [[LOCAL_RESULT]]

// CHECK-LABEL: define void @main.sortLocalToUnnamed(){{.*}} {
// CHECK: call void @qsort({{.*}}ptr @"main.sortLocalToUnnamed$1")
// CHECK-LABEL: define i32 @"main.sortLocalToUnnamed$1"(ptr %0, ptr %1){{.*}} {
// CHECK: sub i64
// CHECK: trunc i64 {{%[0-9]+}} to i32

// CHECK-LABEL: define void @main.sortUnnamedToImported(){{.*}} {
// CHECK: call void @qsort({{.*}}ptr @"main.sortUnnamedToImported$1")
// CHECK-LABEL: define i32 @"main.sortUnnamedToImported$1"(ptr %0, ptr %1){{.*}} {
// CHECK: sub i64
// CHECK: trunc i64 {{%[0-9]+}} to i32
func main() {
	sortLocalNamed()
	sortLocalToUnnamed()
	sortUnnamedToImported()
	sortExplicitNamedConversion()
}

// Local named callback passed to a local named C declaration.
func sortLocalNamed() {
	a := [...]int{100, 8, 23, 2, 7}
	var fn Comp = func(a, b unsafe.Pointer) int32 {
		return int32(*(*int)(a) - *(*int)(b))
	}
	qsortLocal(unsafe.Pointer(&a[0]), uintptr(len(a)), unsafe.Sizeof(a[0]), fn)
	zAssertSorted(&a)
	printf(cstr("local named\n"))
}

// A named callback is assignable to an unnamed callback parameter.
func sortLocalToUnnamed() {
	a := [...]int{100, 8, 23, 2, 7}
	var fn Comp = func(a, b unsafe.Pointer) int32 {
		return int32(*(*int)(a) - *(*int)(b))
	}
	qsortUnnamed(unsafe.Pointer(&a[0]), uintptr(len(a)), unsafe.Sizeof(a[0]), fn)
	zAssertSorted(&a)
	printf(cstr("named to unnamed\n"))
}

// An unnamed literal is assignable to an imported named C callback.
func sortUnnamedToImported() {
	a := [...]int{100, 8, 23, 2, 7}
	fn := func(a, b unsafe.Pointer) int32 {
		return int32(*(*int)(a) - *(*int)(b))
	}
	q.Qsort(unsafe.Pointer(&a[0]), uintptr(len(a)), unsafe.Sizeof(a[0]), fn)
	zAssertSorted(&a)
	printf(cstr("unnamed to imported\n"))
}

// Distinct named callback types require an explicit conversion.
func sortExplicitNamedConversion() {
	a := [...]int{100, 8, 23, 2, 7}
	var fn Comp = func(a, b unsafe.Pointer) int32 {
		return int32(*(*int)(a) - *(*int)(b))
	}
	q.Qsort(unsafe.Pointer(&a[0]), uintptr(len(a)), unsafe.Sizeof(a[0]), q.Comp(fn))
	zAssertSorted(&a)
	printf(cstr("explicit named conversion\n"))
}

func zAssertSorted(a *[5]int) {
	if *a != [5]int{2, 7, 8, 23, 100} {
		panic("qsort did not sort")
	}
}
