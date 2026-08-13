// LITTEST
package main

import "github.com/goplus/lib/c"

// CHECK-LABEL: define i32 @main.f(i32 %0){{.*}} {
// CHECK: [[UINT_NEXT:%[0-9]+]] = add i32 %0, 1
// CHECK-NEXT: ret i32 [[UINT_NEXT]]
func f(a c.Uint) c.Uint {
	a++
	return a
}

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: [[UINT_RESULT:%[0-9]+]] = call i32 @main.f(i32 100)
// CHECK-NEXT: call i32 (ptr, ...) @printf(ptr @{{[0-9]+}}, i32 [[UINT_RESULT]])
func main() {
	var a c.Uint = 100
	c.Printf(c.Str("Hello, %u\n"), f(a))
}
