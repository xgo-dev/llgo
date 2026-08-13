// LITTEST
package main

import (
	"github.com/goplus/lib/c"
)

// CHECK-LABEL: define void @main.callback(ptr %0, { ptr, ptr } %1){{.*}} {
// CHECK: [[CALLBACK_ENV:%[0-9]+]] = extractvalue { ptr, ptr } %1, 1
// CHECK-NEXT: [[CALLBACK_CODE:%[0-9]+]] = extractvalue { ptr, ptr } %1, 0
// CHECK: call void %__llgo_funcval_code(ptr {{(nest|swiftself)}} [[CALLBACK_ENV]], ptr %0)
func callback(msg *c.Char, f func(*c.Char)) {
	f(msg)
}

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: call void @main.callback(ptr @{{[0-9]+}}, { ptr, ptr } { ptr @main.print, ptr null })
// CHECK-NEXT: call void @main.callback(ptr @{{[0-9]+}}, { ptr, ptr } { ptr @main.print, ptr null })
func main() {
	callback(c.Str("Hello\n"), print)
	callback(c.Str("callback\n"), print)
}

// CHECK-LABEL: define void @main.print(ptr %0){{.*}} {
// CHECK: call i32 (ptr, ...) @printf(ptr %0)

func print(msg *c.Char) {
	c.Printf(msg)
}
