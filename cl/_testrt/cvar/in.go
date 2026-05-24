// LITTEST
package main

import _ "unsafe"

//go:linkname barX _bar_x

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/cvar.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testrt/cvar.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testrt/cvar.init$guard", align 1
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

var barX struct {
	Arr       [16]int8
	Callbacks [2]func()
}

//
//go:linkname barY _bar_y
var barY struct {
	Arr [16]int8
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/cvar.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load { [16 x i8], [2 x { ptr, ptr }] }, ptr @"{{.*}}/cl/_testrt/cvar.barX", align 8
// CHECK-NEXT:   %1 = load { [16 x i8] }, ptr @_bar_y, align 1
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

func main() {
	_ = barX
	_ = barY
}
