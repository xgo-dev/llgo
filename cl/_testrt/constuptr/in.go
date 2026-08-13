// LITTEST
package main

import (
	"unsafe"
)

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK:   call void @"{{.*}}/runtime/internal/runtime.PrintPointer"(ptr inttoptr (i64 100 to ptr))
func main() {
	a := unsafe.Pointer(uintptr(100))
	println(a)
}
