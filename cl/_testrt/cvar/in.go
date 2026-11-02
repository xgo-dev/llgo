// LITTEST darwin/arm64 linux/amd64 windows/386 windows/amd64 windows/arm64
// Scope: common (native global storage, including function-pointer fields)
package main

import _ "unsafe"

// CHECK: {{^}}@_bar_x = external global { [16 x i8], [2 x ptr] }
//
//go:linkname barX C._bar_x
var barX struct {
	Arr       [16]int8
	Callbacks [2]func()
}

// CHECK: {{^}}@_bar_y = external global { [16 x i8] }
//
//go:linkname barY C._bar_y
var barY struct {
	Arr [16]int8
}

func main() {
	_ = barX
	_ = barY
}
