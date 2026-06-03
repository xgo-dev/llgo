// LITTEST
package main

import (
	"github.com/goplus/lib/c"
)

// CHECK-LABEL: define void @"{{.*}}/cl/_testlibc/argv.main"(){{.*}} {
func main() {
	// CHECK-NEXT: _llgo_0:
	// CHECK-NEXT:   br label %_llgo_1
	// CHECK-EMPTY:
	// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_2, %_llgo_0
	// CHECK-NEXT:   %0 = phi i32 [ 0, %_llgo_0 ], [ %7, %_llgo_2 ]
	// CHECK-NEXT:   %1 = load i32, ptr @__llgo_argc, align 4
	// CHECK-NEXT:   %2 = icmp slt i32 %0, %1
	// CHECK-NEXT:   br i1 %2, label %_llgo_2, label %_llgo_3
	// CHECK-EMPTY:
	// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1
	// CHECK-NEXT:   %3 = load ptr, ptr @__llgo_argv, align 8
	// CHECK-NEXT:   %4 = getelementptr ptr, ptr %3, i32 %0
	// CHECK-NEXT:   %5 = load ptr, ptr %4, align 8
	// CHECK-NEXT:   %6 = call i32 (ptr, ...) @printf(ptr @0, ptr %5)
	// CHECK-NEXT:   %7 = add i32 %0, 1
	// CHECK-NEXT:   br label %_llgo_1
	// CHECK-EMPTY:
	// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_1
	// CHECK-NEXT:   ret void
	// CHECK-NEXT: }
	for i := c.Int(0); i < c.Argc; i++ {
		c.Printf(c.Str("%s\n"), c.Index(c.Argv, i))
	}
}
