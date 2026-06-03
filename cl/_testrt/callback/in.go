// LITTEST
package main

import (
	"github.com/goplus/lib/c"
)

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/callback.callback"(ptr %0, { ptr, ptr } %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = extractvalue { ptr, ptr } %1, 1
// CHECK-NEXT:   %3 = extractvalue { ptr, ptr } %1, 0
// CHECK-NEXT:   call void %3(ptr %2, ptr %0)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func callback(msg *c.Char, f func(*c.Char)) {
	f(msg)
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/callback.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/callback.callback"(ptr @0, { ptr, ptr } { ptr @"__llgo_stub.github.com/goplus/llgo/cl/_testrt/callback.print", ptr null })
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/callback.callback"(ptr @1, { ptr, ptr } { ptr @"__llgo_stub.github.com/goplus/llgo/cl/_testrt/callback.print", ptr null })
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func main() {
	callback(c.Str("Hello\n"), print)
	callback(c.Str("callback\n"), print)
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/callback.print"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = call i32 (ptr, ...) @printf(ptr %0)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce void @"__llgo_stub.github.com/goplus/llgo/cl/_testrt/callback.print"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   tail call void @"{{.*}}/cl/_testrt/callback.print"(ptr %1)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func print(msg *c.Char) {
	c.Printf(msg)
}
