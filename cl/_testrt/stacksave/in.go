// LITTEST
package main

import (
	"unsafe"
	_ "unsafe"
)

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: [[STACK_POINTER:%[0-9]+]] = call ptr @llvm.stacksave.p0()
// CHECK-NEXT: call void @"{{.*}}/runtime/internal/runtime.PrintPointer"(ptr [[STACK_POINTER]])

//go:linkname getsp llgo.stackSave
func getsp() unsafe.Pointer

func main() {
	sp := getsp()
	println(sp)
}
