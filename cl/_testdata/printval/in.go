// LITTEST
package main

import _ "unsafe"

//go:linkname printf C.printf
func printf(format *int8, __llgo_va_list ...any)

var format = [...]int8{'H', 'e', 'l', 'l', 'o', ' ', '%', 'd', '\n', 0}

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK:   call void (ptr, ...) @printf(ptr @main.format, i64 100)
func main() {
	printf(&format[0], 100)
}
