// LITTEST
package main

import _ "unsafe"

//go:linkname asm llgo.asm
func asm(instruction string)

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK:   call void asm sideeffect "nop", ""()
func main() {
	asm("nop")
}
