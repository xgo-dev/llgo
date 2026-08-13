// LITTEST
package main

import "unsafe"

//
//go:linkname cstr llgo.cstr
func cstr(string) *int8

//go:linkname stderr __stderrp
var stderr unsafe.Pointer

//go:linkname fprintf C.fprintf
func fprintf(fp unsafe.Pointer, format *int8, __llgo_va_list ...any)

// CHECK: [[FPRINTF_FORMAT:@[0-9]+]] = private unnamed_addr constant [10 x i8] c"Hello %d\0A\00"
// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: [[STDERR:%[0-9]+]] = load ptr, ptr @__stderrp
// CHECK-NEXT: call void (ptr, ptr, ...) @fprintf(ptr [[STDERR]], ptr [[FPRINTF_FORMAT]], i64 100)
func main() {
	fprintf(stderr, cstr("Hello %d\n"), 100)
}
