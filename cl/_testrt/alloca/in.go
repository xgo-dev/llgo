// LITTEST
package main

import (
	"github.com/goplus/lib/c"
)

// CHECK: {{^}}@0 = private unnamed_addr constant [4 x i8] c"Hi\0A\00", align 1{{$}}
// CHECK: {{^}}@1 = private unnamed_addr constant [3 x i8] c"%s\00", align 1{{$}}
// CHECK-LABEL: define void @main.main(){{.*}} {
func main() {
	s := c.Str("Hi\n")
	s2 := c.Alloca(4)
	// CHECK: [[ALLOCA_BUF:%[0-9]+]] = alloca i8, i64 4, align 1
	// CHECK-NEXT: [[ALLOCA_COPY:%[0-9]+]] = call ptr @memcpy(ptr [[ALLOCA_BUF]], ptr @0, i64 4)
	// CHECK-NEXT: [[ALLOCA_PRINT:%[0-9]+]] = call i32 (ptr, ...) @printf(ptr @1, ptr [[ALLOCA_BUF]])
	// CHECK-NEXT: ret void
	c.Memcpy(s2, c.Pointer(s), 4)
	c.Printf(c.Str("%s"), s2)
}
