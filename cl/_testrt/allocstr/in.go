// LITTEST
package main

import (
	"github.com/goplus/lib/c"
)

// CHECK: [[HELLO_TEXT:@[0-9]+]] = private unnamed_addr constant [12 x i8] c"Hello world\0A"
// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.String" @main.hello(){{.*}} {
// CHECK: ret %"{{.*}}/runtime/internal/runtime.String" { ptr [[HELLO_TEXT]], i64 12 }
func hello() string {
	return "Hello world\n"
}

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: [[TEXT:%[0-9]+]] = call %"{{.*}}String" @main.hello()
// CHECK-NEXT: [[TEXT_LEN:%[0-9]+]] = extractvalue %"{{.*}}String" [[TEXT]], 1
// CHECK-NEXT: [[CSTR_LEN:%[0-9]+]] = add i64 [[TEXT_LEN]], 1
// CHECK-NEXT: [[CSTR_BUF:%[0-9]+]] = alloca i8, i64 [[CSTR_LEN]]
// CHECK-NEXT: [[CSTR:%[0-9]+]] = call ptr @"{{.*}}CStrCopy"(ptr [[CSTR_BUF]], %"{{.*}}String" [[TEXT]])
// CHECK-NEXT: call i32 (ptr, ...) @printf(ptr [[CSTR]])
func main() {
	c.Printf(c.AllocaCStr(hello()))
}
