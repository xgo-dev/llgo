// LITTEST
package main

import "github.com/goplus/lib/c"

// CHECK: @"{{.*}}/uint.init$guard" = global i1 false, align 1
// CHECK: @0 = private unnamed_addr constant [11 x i8] c"Hello, %u\0A\00", align 1

// CHECK-LABEL: define i32 @"{{.*}}.f"(i32 %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = add i32 %0, 1
// CHECK-NEXT:   ret i32 %1
// CHECK-NEXT: }
func f(a c.Uint) c.Uint {
	a++
	return a
}

// CHECK-LABEL: define void @"{{.*}}.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call i32 @"{{.*}}.f"(i32 100)
// CHECK-NEXT:   %1 = call i32 (ptr, ...) @printf(ptr @0, i32 %0)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func main() {
	var a c.Uint = 100
	c.Printf(c.Str("Hello, %u\n"), f(a))
}
