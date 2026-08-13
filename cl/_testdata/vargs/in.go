// LITTEST
package main

import "github.com/goplus/lib/c"

func main() {
	test(1, 2, 3)
}

func test(a ...any) {
	for _, v := range a {
		c.Printf(c.Str("%d\n"), v.(int))
	}
}

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: [[ARGS_DATA:%.*]] = call ptr @"{{.*}}AllocZ"(i64 48)
// CHECK: [[ARG0_SLOT:%.*]] = getelementptr inbounds %"{{.*}}eface", ptr [[ARGS_DATA]], i64 0
// CHECK: [[ARG0_DATA:%.*]] = call ptr @"{{.*}}AllocU"(i64 8)
// CHECK-NEXT: store i64 1, ptr [[ARG0_DATA]]
// CHECK-NEXT: [[ARG0:%.*]] = insertvalue %"{{.*}}eface" { ptr @_llgo_int, ptr undef }, ptr [[ARG0_DATA]], 1
// CHECK-NEXT: store %"{{.*}}eface" [[ARG0]], ptr [[ARG0_SLOT]]
// CHECK: [[ARG1_SLOT:%.*]] = getelementptr inbounds %"{{.*}}eface", ptr [[ARGS_DATA]], i64 1
// CHECK: [[ARG1_DATA:%.*]] = call ptr @"{{.*}}AllocU"(i64 8)
// CHECK-NEXT: store i64 2, ptr [[ARG1_DATA]]
// CHECK-NEXT: [[ARG1:%.*]] = insertvalue %"{{.*}}eface" { ptr @_llgo_int, ptr undef }, ptr [[ARG1_DATA]], 1
// CHECK-NEXT: store %"{{.*}}eface" [[ARG1]], ptr [[ARG1_SLOT]]
// CHECK: [[ARG2_SLOT:%.*]] = getelementptr inbounds %"{{.*}}eface", ptr [[ARGS_DATA]], i64 2
// CHECK: [[ARG2_DATA:%.*]] = call ptr @"{{.*}}AllocU"(i64 8)
// CHECK-NEXT: store i64 3, ptr [[ARG2_DATA]]
// CHECK-NEXT: [[ARG2:%.*]] = insertvalue %"{{.*}}eface" { ptr @_llgo_int, ptr undef }, ptr [[ARG2_DATA]], 1
// CHECK-NEXT: store %"{{.*}}eface" [[ARG2]], ptr [[ARG2_SLOT]]
// CHECK: [[ARGS0:%.*]] = insertvalue %"{{.*}}Slice" undef, ptr [[ARGS_DATA]], 0
// CHECK-NEXT: [[ARGS1:%.*]] = insertvalue %"{{.*}}Slice" [[ARGS0]], i64 3, 1
// CHECK-NEXT: [[ARGS:%.*]] = insertvalue %"{{.*}}Slice" [[ARGS1]], i64 3, 2
// CHECK-NEXT: call void @main.test(%"{{.*}}Slice" [[ARGS]])

// CHECK-LABEL: define void @main.test(%"{{.*}}Slice" %0){{.*}} {
// CHECK: [[ARGS_LEN:%.*]] = extractvalue %"{{.*}}Slice" %0, 1
// CHECK: [[I0:%.*]] = phi i64 [ -1, %{{.*}} ], [ [[I:%.*]], %{{.*}} ]
// CHECK-NEXT: [[I]] = add i64 [[I0]], 1
// CHECK-NEXT: [[MORE:%.*]] = icmp slt i64 [[I]], [[ARGS_LEN]]
// CHECK: [[DATA:%.*]] = extractvalue %"{{.*}}Slice" %0, 0
// CHECK: [[BOUNDS_LEN:%.*]] = extractvalue %"{{.*}}Slice" %0, 1
// CHECK: call void @"{{.*}}CheckIndexRange"(i1 {{%.*}}, i64 [[I]], i1 true, i64 [[BOUNDS_LEN]])
// CHECK-NEXT: [[ARG_SLOT:%.*]] = getelementptr inbounds %"{{.*}}eface", ptr [[DATA]], i64 [[I]]
// CHECK-NEXT: [[ARG:%.*]] = load %"{{.*}}eface", ptr [[ARG_SLOT]]
// CHECK-NEXT: [[ARG_TYPE:%.*]] = extractvalue %"{{.*}}eface" [[ARG]], 0
// CHECK-NEXT: [[IS_INT:%.*]] = icmp eq ptr [[ARG_TYPE]], @_llgo_int
// CHECK-NEXT: br i1 [[IS_INT]], label %{{.*}}, label %{{.*}}
// CHECK: [[ARG_DATA:%.*]] = extractvalue %"{{.*}}eface" [[ARG]], 1
// CHECK-NEXT: [[ARG_VALUE:%.*]] = load i64, ptr [[ARG_DATA]]
// CHECK-NEXT: call i32 (ptr, ...) @printf(ptr @{{[0-9]+}}, i64 [[ARG_VALUE]])
// CHECK: call void @"{{.*}}PanicTypeAssert"(ptr null, ptr [[ARG_TYPE]], ptr @_llgo_int)
// CHECK-NEXT: unreachable
