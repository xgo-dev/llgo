// LITTEST
package main

import (
	"github.com/goplus/lib/c"
)

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/unreachable.foo"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   unreachable
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func foo() {
	c.Unreachable()
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/unreachable.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/unreachable.foo"()
// CHECK-NEXT:   %0 = call i32 (ptr, ...) @printf(ptr @0)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func main() {
	foo()
	c.Printf(c.Str("Hello\n"))
}
