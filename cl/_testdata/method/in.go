// LITTEST
package main

import _ "unsafe"

type T int

// CHECK-LABEL: define i64 @main.T.Add(i64 %0, i64 %1){{.*}} {
// CHECK: [[SUM:%[0-9]+]] = add i64 %0, %1
// CHECK-NEXT: ret i64 [[SUM]]

func (a T) Add(b T) T {
	return a + b
}

//go:linkname printf C.printf
func printf(format *int8, __llgo_va_list ...any)

var format = [...]int8{'H', 'e', 'l', 'l', 'o', ' ', '%', 'd', '\n', 0}

func main() {
	a := T(1)
	printf(&format[0], a.Add(2))
}

// CHECK-LABEL: define i64 @"main.(*T).Add"(ptr %0, i64 %1){{.*}} {
// CHECK: [[NIL:%[0-9]+]] = icmp eq ptr %0, null
// CHECK-NEXT: call void @"{{.*}}PanicWrapNilPointer"(i1 [[NIL]], %"{{.*}}String" { ptr @{{[0-9]+}}, i64 44 }, %"{{.*}}String" { ptr @{{[0-9]+}}, i64 3 })
// CHECK-NEXT: [[RECEIVER:%[0-9]+]] = load i64, ptr %0
// CHECK-NEXT: [[WRAPPED_SUM:%[0-9]+]] = call i64 @main.T.Add(i64 [[RECEIVER]], i64 %1)
// CHECK-NEXT: ret i64 [[WRAPPED_SUM]]

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: [[MAIN_SUM:%[0-9]+]] = call i64 @main.T.Add(i64 1, i64 2)
// CHECK-NEXT: call void (ptr, ...) @printf(ptr @main.format, i64 [[MAIN_SUM]])
