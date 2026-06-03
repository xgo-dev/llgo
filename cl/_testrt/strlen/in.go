// LITTEST
package main

import "C"
import _ "unsafe"

//go:linkname printf C.printf
func printf(format *int8, __llgo_va_list ...any)

//go:linkname strlen C.strlen
func strlen(str *int8) C.int

var format = [...]int8{'H', 'e', 'l', 'l', 'o', ' ', '%', 'd', '\n', 0}

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/strlen.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call i32 @strlen(ptr @"{{.*}}/cl/_testrt/strlen.format")
// CHECK-NEXT:   call void (ptr, ...) @printf(ptr @"{{.*}}/cl/_testrt/strlen.format", i32 %0)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func main() {
	sfmt := &format[0]
	printf(sfmt, strlen(sfmt))
}
