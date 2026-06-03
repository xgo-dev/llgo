// LITTEST
package main

import (
	"unsafe"
	_ "unsafe"
)

//go:linkname getsp llgo.stackSave
func getsp() unsafe.Pointer

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/stacksave.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @llvm.stacksave()
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintPointer"(ptr %0)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func main() {
	sp := getsp()
	println(sp)
}
