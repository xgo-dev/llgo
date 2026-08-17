// LITTEST
package main

import "github.com/xgo-dev/llgo/cl/_testrt/hello/libc"

// CHECK: {{^}}@main.format = global [10 x i8] c"Hello %d\0A\00"

var format = [...]int8{'H', 'e', 'l', 'l', 'o', ' ', '%', 'd', '\n', 0}

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: [[LEN:%[0-9]+]] = call i32 @strlen(ptr @main.format)
// CHECK-NEXT: call void (ptr, ...) @printf(ptr @main.format, i32 [[LEN]])
func main() {
	sfmt := &format[0]
	libc.Printf(sfmt, libc.Strlen(sfmt))
}
