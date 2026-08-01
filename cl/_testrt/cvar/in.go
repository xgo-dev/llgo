// LITTEST
package main

import _ "unsafe"

// CHECK: {{^}}@_bar_x = external global { [16 x i8], [2 x ptr] }, align 8{{$}}
//
//go:linkname barX _bar_x
var barX struct {
	Arr       [16]int8
	Callbacks [2]func()
}

// CHECK: {{^}}@_bar_y = external global { [16 x i8] }, align 1{{$}}
//
//go:linkname barY _bar_y
var barY struct {
	Arr [16]int8
}

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func main() {
	_ = barX
	_ = barY
}
