// LITTEST
package main

import (
	"github.com/goplus/llgo/cl/_testgo/runextest/bar"
	"github.com/goplus/llgo/cl/_testgo/runextest/foo"
)

// Check that the external test packages participate in initialization and
// remain direct calls from the executable package.
// CHECK-LABEL: define i64 @main.Zoo(){{.*}} {
// CHECK: ret i64 3
// CHECK-LABEL: define void @main.init(){{.*}} {
// CHECK: call void @"{{.*}}/runextest/bar.init"()
// CHECK-NEXT: call void @"{{.*}}/runextest/foo.init"()
// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: [[EXT_FOO:%[0-9]+]] = call i64 @"{{.*}}/runextest/foo.Foo"()
// CHECK: call void @"{{.*}}PrintInt"(i64 [[EXT_FOO]])
// CHECK: [[EXT_BAR:%[0-9]+]] = call i64 @"{{.*}}/runextest/bar.Bar"()
// CHECK: call void @"{{.*}}PrintInt"(i64 [[EXT_BAR]])
// CHECK: [[EXT_ZOO:%[0-9]+]] = call i64 @main.Zoo()
// CHECK: call void @"{{.*}}PrintInt"(i64 [[EXT_ZOO]])

func Zoo() int {
	return 3
}

func main() {
	println("foo.Foo()", foo.Foo())
	println("bar.Bar()", bar.Bar())
	println("Zoo()", Zoo())
}
