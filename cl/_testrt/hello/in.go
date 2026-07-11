// LITTEST
package main

import "github.com/goplus/llgo/cl/_testrt/hello/libc"

// CHECK: {{^}}@"{{.*}}/cl/_testrt/hello.format" = global [10 x i8] c"Hello %d\0A\00", align 1{{$}}

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/hello.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testrt/hello.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testrt/hello.init$guard", align 1
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
var format = [...]int8{'H', 'e', 'l', 'l', 'o', ' ', '%', 'd', '\n', 0}

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/hello.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call i32 @strlen(ptr @"{{.*}}/cl/_testrt/hello.format")
// CHECK-NEXT:   call void (ptr, ...) @printf(ptr @"{{.*}}/cl/_testrt/hello.format", i32 %0)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func main() {
	sfmt := &format[0]
	libc.Printf(sfmt, libc.Strlen(sfmt))
}
