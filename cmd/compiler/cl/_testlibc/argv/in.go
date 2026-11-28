// LITTEST
// Scope: common
package main

import _ "unsafe"

type integer interface{ ~int32 }

//go:linkname argc C.__llgo_argc
var argc int32

//go:linkname argv C.__llgo_argv
var argv **int8

// llgo:link index llgo.index
func index[T any, I integer](ptr *T, offset I) T { return *ptr }

//go:linkname cstr llgo.cstr
func cstr(string) *int8

//go:linkname printf C.printf
func printf(format *int8, __llgo_va_list ...any) int32

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
	for i := int32(0); i < argc; i++ {
		printf(cstr("%s\n"), index(argv, i))
	}
}
