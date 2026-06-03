// LITTEST
package main

import (
	"github.com/goplus/lib/c"
)

// CHECK-LINE: @0 = private unnamed_addr constant [4 x i8] c"Hi\0A\00", align 1
// CHECK-LINE: @1 = private unnamed_addr constant [3 x i8] c"%s\00", align 1
// CHECK-LABEL: define void @"{{.*}}alloca.main"(){{.*}} {
func main() {
	s := c.Str("Hi\n")
	s2 := c.Alloca(4)
	// CHECK: alloca i8, i64 4, align 1
	// CHECK-NEXT: @memcpy(ptr {{%[0-9]+}}, ptr @0, i64 4)
	// CHECK-NEXT: @printf(ptr @1, ptr {{%[0-9]+}})
	// CHECK-NEXT: ret void
	c.Memcpy(s2, c.Pointer(s), 4)
	c.Printf(c.Str("%s"), s2)
}
