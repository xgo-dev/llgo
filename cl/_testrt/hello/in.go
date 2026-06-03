// LITTEST
package main

import "github.com/goplus/llgo/cl/_testrt/hello/libc"

// CHECK-LINE: @"{{.*}}/cl/_testrt/hello.format" = global [10 x i8] zeroinitializer, align 1

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/hello.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testrt/hello.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testrt/hello.init$guard", align 1
// CHECK-NEXT:   store i8 72, ptr @"{{.*}}/cl/_testrt/hello.format", align 1
// CHECK-NEXT:   store i8 101, ptr getelementptr inbounds (i8, ptr @"{{.*}}/cl/_testrt/hello.format", i64 1), align 1
// CHECK-NEXT:   store i8 108, ptr getelementptr inbounds (i8, ptr @"{{.*}}/cl/_testrt/hello.format", i64 2), align 1
// CHECK-NEXT:   store i8 108, ptr getelementptr inbounds (i8, ptr @"{{.*}}/cl/_testrt/hello.format", i64 3), align 1
// CHECK-NEXT:   store i8 111, ptr getelementptr inbounds (i8, ptr @"{{.*}}/cl/_testrt/hello.format", i64 4), align 1
// CHECK-NEXT:   store i8 32, ptr getelementptr inbounds (i8, ptr @"{{.*}}/cl/_testrt/hello.format", i64 5), align 1
// CHECK-NEXT:   store i8 37, ptr getelementptr inbounds (i8, ptr @"{{.*}}/cl/_testrt/hello.format", i64 6), align 1
// CHECK-NEXT:   store i8 100, ptr getelementptr inbounds (i8, ptr @"{{.*}}/cl/_testrt/hello.format", i64 7), align 1
// CHECK-NEXT:   store i8 10, ptr getelementptr inbounds (i8, ptr @"{{.*}}/cl/_testrt/hello.format", i64 8), align 1
// CHECK-NEXT:   store i8 0, ptr getelementptr inbounds (i8, ptr @"{{.*}}/cl/_testrt/hello.format", i64 9), align 1
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
