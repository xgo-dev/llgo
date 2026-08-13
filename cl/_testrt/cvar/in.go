// LITTEST
package main

import _ "unsafe"

// CHECK: {{^}}@_bar_x = external global { [16 x i8], [2 x ptr] }
//
//go:linkname barX _bar_x
var barX struct {
	Arr       [16]int8
	Callbacks [2]func()
}

// CHECK: {{^}}@_bar_y = external global { [16 x i8] }
//
//go:linkname barY _bar_y
var barY struct {
	Arr [16]int8
}

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: load { [16 x i8], [2 x ptr] }, ptr @_bar_x
// CHECK: load { [16 x i8] }, ptr @_bar_y
func main() {
	_ = barX
	_ = barY
}
