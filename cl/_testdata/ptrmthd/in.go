// LITTEST
package main

import _ "unsafe"

//go:linkname printf C.printf
func printf(format *int8, __llgo_va_list ...any)

type T int8

// CHECK-LABEL: define void @"main.(*T).Print"(ptr %0, i64 %1){{.*}} {
// CHECK: call void (ptr, ...) @printf(ptr %0, i64 %1)
// CHECK-NEXT: ret void
func (f *T) Print(v int) {
	printf((*int8)(f), v)
}

var format = [...]T{'H', 'e', 'l', 'l', 'o', ' ', '%', 'd', '\n', 0}

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: call void @"main.(*T).Print"(ptr @main.format, i64 100)
func main() {
	f := &format[0]
	f.Print(100)
}
