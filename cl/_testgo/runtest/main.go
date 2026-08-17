// LITTEST
package main

import (
	"github.com/xgo-dev/llgo/cl/_testgo/runtest/bar"
	"github.com/xgo-dev/llgo/cl/_testgo/runtest/foo"
)

// CHECK-LABEL: define i64 @main.Zoo(){{.*}} {
// CHECK: ret i64 3
// CHECK-LABEL: define void @main.init(){{.*}} {
// CHECK: call void @"{{.*}}/runtest/bar.init"()
// CHECK-NEXT: call void @"{{.*}}/runtest/foo.init"()
// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: [[FOO:%[0-9]+]] = call i64 @"{{.*}}/runtest/foo.Foo"()
// CHECK: call void @"{{.*}}PrintInt"(i64 [[FOO]])
// CHECK: [[BAR:%[0-9]+]] = call i64 @"{{.*}}/runtest/bar.Bar"()
// CHECK: call void @"{{.*}}PrintInt"(i64 [[BAR]])
// CHECK: [[ZOO:%[0-9]+]] = call i64 @main.Zoo()
// CHECK: call void @"{{.*}}PrintInt"(i64 [[ZOO]])

func Zoo() int {
	return 3
}

func main() {
	println("foo.Foo()", foo.Foo())
	println("bar.Bar()", bar.Bar())
	println("Zoo()", Zoo())
}
