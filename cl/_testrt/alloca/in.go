// LITTEST
package main

import (
	"github.com/goplus/lib/c"
)

// CHECK-LINE: @0 = private unnamed_addr constant [4 x i8] c"Hi\0A\00", align 1
// CHECK-LINE: @1 = private unnamed_addr constant [3 x i8] c"%s\00", align 1

func main() {
	s := c.Str("Hi\n")
	s2 := c.Alloca(4)
	c.Memcpy(s2, c.Pointer(s), 4)
	c.Printf(c.Str("%s"), s2)
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/alloca.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testrt/alloca.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testrt/alloca.init$guard", align 1
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/alloca.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = alloca i8, i64 4, align 1
// CHECK-NEXT:   %1 = call ptr @memcpy(ptr %0, ptr @0, i64 4)
// CHECK-NEXT:   %2 = call i32 (ptr, ...) @printf(ptr @1, ptr %0)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
