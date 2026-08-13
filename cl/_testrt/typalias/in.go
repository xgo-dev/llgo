// LITTEST
package main

import "C"
import _ "unsafe"

//go:linkname printf C.printf
func printf(format *int8, __llgo_va_list ...any)

type Foo = struct {
	A  C.int
	ok bool
}

var format = [...]int8{'H', 'e', 'l', 'l', 'o', ' ', '%', 'd', '\n', 0}

// CHECK-LABEL: define void @main.Print(ptr %0){{.*}} {
// CHECK: [[PRINT_OK_FIELD:%[0-9]+]] = getelementptr inbounds { i32, i1 }, ptr %0, i32 0, i32 1
// CHECK-NEXT: [[PRINT_OK:%[0-9]+]] = load i1, ptr [[PRINT_OK_FIELD]]
// CHECK-NEXT: br i1 [[PRINT_OK]], label %{{[^,]+}}, label %{{[^ ]+}}
// CHECK: [[PRINT_A_FIELD:%[0-9]+]] = getelementptr inbounds { i32, i1 }, ptr %0, i32 0, i32 0
// CHECK-NEXT: [[PRINT_A:%[0-9]+]] = load i32, ptr [[PRINT_A_FIELD]]
// CHECK-NEXT: call void (ptr, ...) @printf(ptr @main.format, i32 [[PRINT_A]])

func Print(p *Foo) {
	if p.ok {
		printf(&format[0], p.A)
	}
}

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: [[MAIN_FOO:%[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT: [[MAIN_A:%[0-9]+]] = getelementptr inbounds { i32, i1 }, ptr [[MAIN_FOO]], i32 0, i32 0
// CHECK-NEXT: [[MAIN_OK:%[0-9]+]] = getelementptr inbounds { i32, i1 }, ptr [[MAIN_FOO]], i32 0, i32 1
// CHECK-NEXT: store i32 100, ptr [[MAIN_A]]
// CHECK-NEXT: store i1 true, ptr [[MAIN_OK]]
// CHECK-NEXT: call void @main.Print(ptr [[MAIN_FOO]])
func main() {
	foo := &Foo{100, true}
	Print(foo)
}
