// LITTEST
package main

import (
	"github.com/goplus/lib/c"
)

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: [[ARGV_INDEX:%[0-9]+]] = phi i32 [ 0, %{{.*}} ], [ %{{[0-9]+}}, %{{.*}} ]
// CHECK: [[ARGV_COUNT:%[0-9]+]] = load i32, ptr @__llgo_argc
// CHECK-NEXT: [[ARGV_MORE:%[0-9]+]] = icmp slt i32 [[ARGV_INDEX]], [[ARGV_COUNT]]
// CHECK-NEXT: br i1 [[ARGV_MORE]], label %{{.*}}, label %{{.*}}
// CHECK: [[ARGV_BASE:%[0-9]+]] = load ptr, ptr @__llgo_argv
// CHECK-NEXT: [[ARGV_SLOT:%[0-9]+]] = getelementptr ptr, ptr [[ARGV_BASE]], i32 [[ARGV_INDEX]]
// CHECK-NEXT: [[ARGV_VALUE:%[0-9]+]] = load ptr, ptr [[ARGV_SLOT]]
// CHECK-NEXT: call i32 (ptr, ...) @printf(ptr @{{[0-9]+}}, ptr [[ARGV_VALUE]])
// CHECK: [[ARGV_NEXT:%[0-9]+]] = add i32 [[ARGV_INDEX]], 1

func main() {
	for i := c.Int(0); i < c.Argc; i++ {
		c.Printf(c.Str("%s\n"), c.Index(c.Argv, i))
	}
}
