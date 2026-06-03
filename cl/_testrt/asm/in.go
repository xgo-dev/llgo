// LITTEST
package main

import _ "unsafe"

//go:linkname asm llgo.asm
func asm(instruction string)

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/asm.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   call void asm sideeffect "nop", ""()
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func main() {
	asm("nop")
}
