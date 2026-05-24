// LITTEST
package main

import (
	"unsafe"

	"github.com/goplus/lib/c"
)

// CHECK-LINE: @0 = private unnamed_addr constant [3 x i8] c"%d\00", align 1

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/structsize.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testrt/structsize.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testrt/structsize.init$guard", align 1
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

type Foo struct {
	A byte
	B uint8
	C uint16
	D byte
	E [8]int8
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/structsize.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call i32 (ptr, ...) @printf(ptr @0, i64 14)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

func main() {
	c.Printf(c.Str("%d"), unsafe.Sizeof(Foo{}))
}
