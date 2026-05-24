// LITTEST
package main

import (
	"github.com/goplus/lib/c"
)

// CHECK-LINE: @0 = private unnamed_addr constant [7 x i8] c"Hello\0A\00", align 1

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/unreachable.foo"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   unreachable
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

func foo() {
	c.Unreachable()
}

func main() {
	foo()
	c.Printf(c.Str("Hello\n"))
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/unreachable.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testrt/unreachable.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testrt/unreachable.init$guard", align 1
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/unreachable.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/unreachable.foo"()
// CHECK-NEXT:   %0 = call i32 (ptr, ...) @printf(ptr @0)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
