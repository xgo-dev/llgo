// LITTEST
package main

// The loop carries both the accumulated string and the current index. The
// element selected by that index must be concatenated into the carried value.
// CHECK-LABEL: define %"{{.*}}String" @main.concat(%"{{.*}}Slice" %0){{.*}} {
// CHECK: [[LEN:%[0-9]+]] = extractvalue %"{{.*}}Slice" %0, 1
// CHECK: [[ACC:%[0-9]+]] = phi %"{{.*}}String" [ zeroinitializer, %{{.*}} ], [ [[NEXT_ACC:%[0-9]+]], %{{.*}} ]
// CHECK-NEXT: [[PREV_INDEX:%[0-9]+]] = phi i64 [ -1, %{{.*}} ], [ [[INDEX:%[0-9]+]], %{{.*}} ]
// CHECK-NEXT: [[INDEX]] = add i64 [[PREV_INDEX]], 1
// CHECK-NEXT: [[MORE:%[0-9]+]] = icmp slt i64 [[INDEX]], [[LEN]]
// CHECK: [[DATA:%[0-9]+]] = extractvalue %"{{.*}}Slice" %0, 0
// CHECK-NEXT: [[BOUNDS_LEN:%[0-9]+]] = extractvalue %"{{.*}}Slice" %0, 1
// CHECK: call void @"{{.*}}CheckIndexRange"(i1 {{.*}}, i64 [[INDEX]], i1 true, i64 [[BOUNDS_LEN]])
// CHECK-NEXT: [[ELEM_ADDR:%[0-9]+]] = getelementptr inbounds %"{{.*}}String", ptr [[DATA]], i64 [[INDEX]]
// CHECK-NEXT: [[ELEM:%[0-9]+]] = load %"{{.*}}String", ptr [[ELEM_ADDR]]
// CHECK-NEXT: [[NEXT_ACC]] = call %"{{.*}}String" @"{{.*}}StringCat"(%"{{.*}}String" [[ACC]], %"{{.*}}String" [[ELEM]])
// CHECK: ret %"{{.*}}String" [[ACC]]
func concat(args ...string) (ret string) {
	for _, v := range args {
		ret += v
	}
	return
}

// CHECK-LABEL: define %"{{.*}}String" @main.info(%"{{.*}}String" %0){{.*}} {
// CHECK: [[INFO_PREFIX:%[0-9]+]] = call %"{{.*}}String" @"{{.*}}StringCat"(%"{{.*}}String" zeroinitializer, %"{{.*}}String" %0)
// CHECK-NEXT: [[INFO_RESULT:%[0-9]+]] = call %"{{.*}}String" @"{{.*}}StringCat"(%"{{.*}}String" [[INFO_PREFIX]], %"{{.*}}String" { ptr @{{[0-9]+}}, i64 3 })
// CHECK-NEXT: ret %"{{.*}}String" [[INFO_RESULT]]
func info(s string) string {
	return "" + s + "..."
}

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: [[ARGS_DATA:%[0-9]+]] = call ptr @"{{.*}}AllocZ"(i64 48)
// CHECK-NEXT: [[ARG0:%[0-9]+]] = getelementptr inbounds %"{{.*}}String", ptr [[ARGS_DATA]], i64 0
// CHECK-NEXT: store %"{{.*}}String" { ptr @{{[0-9]+}}, i64 5 }, ptr [[ARG0]]
// CHECK-NEXT: [[ARG1:%[0-9]+]] = getelementptr inbounds %"{{.*}}String", ptr [[ARGS_DATA]], i64 1
// CHECK-NEXT: store %"{{.*}}String" { ptr @{{[0-9]+}}, i64 1 }, ptr [[ARG1]]
// CHECK-NEXT: [[ARG2:%[0-9]+]] = getelementptr inbounds %"{{.*}}String", ptr [[ARGS_DATA]], i64 2
// CHECK-NEXT: store %"{{.*}}String" { ptr @{{[0-9]+}}, i64 5 }, ptr [[ARG2]]
// CHECK-NEXT: [[ARGS0:%[0-9]+]] = insertvalue %"{{.*}}Slice" undef, ptr [[ARGS_DATA]], 0
// CHECK-NEXT: [[ARGS1:%[0-9]+]] = insertvalue %"{{.*}}Slice" [[ARGS0]], i64 3, 1
// CHECK-NEXT: [[ARGS:%[0-9]+]] = insertvalue %"{{.*}}Slice" [[ARGS1]], i64 3, 2
// CHECK-NEXT: [[RESULT:%[0-9]+]] = call %"{{.*}}String" @main.concat(%"{{.*}}Slice" [[ARGS]])
// CHECK-NEXT: call void @"{{.*}}PrintString"(%"{{.*}}String" [[RESULT]])
func main() {
	result := concat("Hello", " ", "World")
	println(result)
}
