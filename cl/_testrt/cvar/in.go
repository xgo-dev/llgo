// LITTEST
package main

import _ "unsafe"

// CHECK-LINE: @_bar_x = external global { [16 x i8], [2 x ptr] }, align 8
//
//go:linkname barX _bar_x
var barX struct {
	Arr       [16]int8
	Callbacks [2]func()
}

// CHECK-LINE: @_bar_y = external global { [16 x i8] }, align 1
//
//go:linkname barY _bar_y
var barY struct {
	Arr [16]int8
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/cvar.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load { [16 x i8], [2 x ptr] }, ptr @_bar_x, align 8
// CHECK-NEXT:   %1 = load { [16 x i8] }, ptr @_bar_y, align 1
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func main() {
	_ = barX
	_ = barY
}
