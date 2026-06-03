// LITTEST
package main

import _ "unsafe"

// CHECK-LINE: @0 = private unnamed_addr constant [14 x i8] c"Hello, world\0A\00", align 1

//go:linkname cstr llgo.cstr
func cstr(string) *int8

//go:linkname printf C.printf
func printf(format *int8, __llgo_va_list ...any)

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/cstr.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   call void (ptr, ...) @printf(ptr @0)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func main() {
	printf(cstr("Hello, world\n"))
}
