// LITTEST
package main

import "github.com/goplus/llgo/cl/_testdata/importpkg/stdio"

// CHECK: @main.hello = global [7 x i8] c"Hello\0A\00"

// CHECK-LABEL: define void @main.init(){{.*}} {
// CHECK: call void @"{{.*}}/stdio.init"()
var hello = [...]int8{'H', 'e', 'l', 'l', 'o', '\n', 0}

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: call i64 @"{{.*}}/stdio.Max"(i64 2, i64 100)
// CHECK:   call void (ptr, ...) @printf(ptr @main.hello)
func main() {
	_ = stdio.Max(2, 100)
	stdio.Printf(&hello[0])
}
