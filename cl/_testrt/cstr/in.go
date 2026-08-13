// LITTEST
package main

import _ "unsafe"

//go:linkname cstr llgo.cstr
func cstr(string) *int8

//go:linkname printf C.printf
func printf(format *int8, __llgo_va_list ...any)

// CHECK: [[CSTR_TEXT:@[0-9]+]] = private unnamed_addr constant [14 x i8] c"Hello, world\0A\00"
// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: call void (ptr, ...) @printf(ptr [[CSTR_TEXT]])
func main() {
	printf(cstr("Hello, world\n"))
}
