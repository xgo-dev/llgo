// LITTEST
package main

import _ "unsafe"

//go:linkname printf C.printf
func printf(format *int8, __llgo_va_list ...any)

type T int8

// CHECK-LABEL: define void @"{{.*}}.(*T).Print"(ptr %0, i64 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   call void (ptr, ...) @printf(ptr %0, i64 %1)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func (f *T) Print(v int) {
	printf((*int8)(f), v)
}

var format = [...]T{'H', 'e', 'l', 'l', 'o', ' ', '%', 'd', '\n', 0}

// CHECK-LABEL: define void @"{{.*}}.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   call void @"{{.*}}.(*T).Print"(ptr @"{{.*}}.format", i64 100)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func main() {
	f := &format[0]
	f.Print(100)
}
