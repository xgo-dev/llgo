// LITTEST
package main

import (
	"github.com/goplus/lib/c"
)

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: [[ARGS_DATA:%[0-9]+]] = call ptr @"{{.*}}AllocZ"(i64 32)
// CHECK-NEXT: [[SUM_ARG0:%[0-9]+]] = getelementptr inbounds i64, ptr [[ARGS_DATA]], i64 0
// CHECK-NEXT: store i64 1, ptr [[SUM_ARG0]]
// CHECK-NEXT: [[SUM_ARG1:%[0-9]+]] = getelementptr inbounds i64, ptr [[ARGS_DATA]], i64 1
// CHECK-NEXT: store i64 2, ptr [[SUM_ARG1]]
// CHECK-NEXT: [[SUM_ARG2:%[0-9]+]] = getelementptr inbounds i64, ptr [[ARGS_DATA]], i64 2
// CHECK-NEXT: store i64 3, ptr [[SUM_ARG2]]
// CHECK-NEXT: [[SUM_ARG3:%[0-9]+]] = getelementptr inbounds i64, ptr [[ARGS_DATA]], i64 3
// CHECK-NEXT: store i64 4, ptr [[SUM_ARG3]]
// CHECK: [[ARGS0:%[0-9]+]] = insertvalue %"{{.*}}Slice" undef, ptr [[ARGS_DATA]], 0
// CHECK-NEXT: [[ARGS1:%[0-9]+]] = insertvalue %"{{.*}}Slice" [[ARGS0]], i64 4, 1
// CHECK-NEXT: [[ARGS:%[0-9]+]] = insertvalue %"{{.*}}Slice" [[ARGS1]], i64 4, 2
// CHECK-NEXT: [[TOTAL:%[0-9]+]] = call i64 @main.sum(%"{{.*}}Slice" [[ARGS]])
// CHECK-NEXT: call i32 (ptr, ...) @printf(ptr @{{[0-9]+}}, i64 [[TOTAL]])
func main() {
	c.Printf(c.Str("Hello %d\n"), sum(1, 2, 3, 4))
}

// CHECK-LABEL: define i64 @main.sum(%"{{.*}}Slice" %0){{.*}} {
// CHECK: [[SUM_LEN:%[0-9]+]] = extractvalue %"{{.*}}Slice" %0, 1
// CHECK: [[SUM_ACC:%[0-9]+]] = phi i64 [ 0, %{{.*}} ], [ [[SUM_NEXT:%[0-9]+]], %{{.*}} ]
// CHECK-NEXT: [[SUM_PREV_INDEX:%[0-9]+]] = phi i64 [ -1, %{{.*}} ], [ [[SUM_INDEX:%[0-9]+]], %{{.*}} ]
// CHECK-NEXT: [[SUM_INDEX]] = add i64 [[SUM_PREV_INDEX]], 1
// CHECK-NEXT: [[SUM_MORE:%[0-9]+]] = icmp slt i64 [[SUM_INDEX]], [[SUM_LEN]]
// CHECK: [[SUM_DATA:%[0-9]+]] = extractvalue %"{{.*}}Slice" %0, 0
// CHECK-NEXT: [[SUM_BOUNDS_LEN:%[0-9]+]] = extractvalue %"{{.*}}Slice" %0, 1
// CHECK: call void @"{{.*}}CheckIndexRange"(i1 {{.*}}, i64 [[SUM_INDEX]], i1 true, i64 [[SUM_BOUNDS_LEN]])
// CHECK-NEXT: [[SUM_ADDR:%[0-9]+]] = getelementptr inbounds i64, ptr [[SUM_DATA]], i64 [[SUM_INDEX]]
// CHECK-NEXT: [[SUM_VALUE:%[0-9]+]] = load i64, ptr [[SUM_ADDR]]
// CHECK-NEXT: [[SUM_NEXT]] = add i64 [[SUM_ACC]], [[SUM_VALUE]]
// CHECK: ret i64 [[SUM_ACC]]
func sum(args ...int) (ret int) {
	for _, v := range args {
		ret += v
	}
	return
}
