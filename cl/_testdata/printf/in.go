// LITTEST
package main

import _ "unsafe"

//go:linkname printf C.printf
func printf(format *int8, __llgo_va_list ...any)

var hello = [...]int8{'H', 'e', 'l', 'l', 'o', '\n', 0}

// CHECK-LABEL: define void @"{{.*}}.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   call void (ptr, ...) @printf(ptr @"{{.*}}.hello")
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func main() {
	printf(&hello[0])
}
