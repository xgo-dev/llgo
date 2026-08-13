// LITTEST
package main

import "C"
import _ "unsafe"

// CHECK-LABEL: define void @main.Foo.Print(%main.Foo %0){{.*}} {
// CHECK: store %main.Foo %0, ptr [[PRINT_RECEIVER:%[0-9]+]]
// CHECK: [[PRINT_OK_FIELD:%[0-9]+]] = getelementptr inbounds %main.Foo, ptr [[PRINT_RECEIVER]], i32 0, i32 1
// CHECK-NEXT: [[PRINT_OK:%[0-9]+]] = load i1, ptr [[PRINT_OK_FIELD]]
// CHECK-NEXT: br i1 [[PRINT_OK]], label %{{.*}}, label %{{.*}}
// CHECK: [[PRINT_VALUE_FIELD:%[0-9]+]] = getelementptr inbounds %main.Foo, ptr [[PRINT_RECEIVER]], i32 0, i32 0
// CHECK-NEXT: [[PRINT_VALUE:%[0-9]+]] = load i32, ptr [[PRINT_VALUE_FIELD]]
// CHECK-NEXT: call void (ptr, ...) @printf(ptr @main.format, i32 [[PRINT_VALUE]])

func (p Foo) Print() {
	if p.ok {
		printf(&format[0], p.A)
	}
}

//go:linkname printf C.printf
func printf(format *int8, __llgo_va_list ...any)

type Foo struct {
	A  C.int
	ok bool
}

var format = [...]int8{'H', 'e', 'l', 'l', 'o', ' ', '%', 'd', '\n', 0}

func main() {
	foo := Foo{100, true}
	foo.Print()
}

// CHECK-LABEL: define void @"main.(*Foo).Print"(ptr %0){{.*}} {
// CHECK: [[WRAPPER_NIL:%[0-9]+]] = icmp eq ptr %0, null
// CHECK-NEXT: call void @"{{.*}}/runtime/internal/runtime.PanicWrapNilPointer"(i1 [[WRAPPER_NIL]], %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 44 }, %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 5 })
// CHECK-NEXT: [[WRAPPER_VALUE:%[0-9]+]] = load %main.Foo, ptr %0
// CHECK-NEXT: call void @main.Foo.Print(%main.Foo [[WRAPPER_VALUE]])

// CHECK-LABEL: define ptr @main._Cgo_ptr(ptr %0){{.*}} {
// CHECK: ret ptr %0

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: store i32 100, ptr [[MAIN_A:%[0-9]+]]
// CHECK-NEXT: store i1 true, ptr [[MAIN_OK:%[0-9]+]]
// CHECK-NEXT: [[MAIN_FOO:%[0-9]+]] = load %main.Foo, ptr %{{[0-9]+}}
// CHECK-NEXT: call void @main.Foo.Print(%main.Foo [[MAIN_FOO]])
