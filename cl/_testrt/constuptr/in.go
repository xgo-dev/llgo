// LITTEST
package main

import (
	"unsafe"
)

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/constuptr.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintPointer"(ptr inttoptr (i64 100 to ptr))
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func main() {
	a := unsafe.Pointer(uintptr(100))
	println(a)
}
