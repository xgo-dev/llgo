// LITTEST
package main

// The condition is evaluated before installing its selected defer, while all
// paths share the same defer-chain cleanup.
// CHECK: @[[HELLO:[0-9]+]] = private unnamed_addr constant [5 x i8] c"hello"
// CHECK: @[[BYE:[0-9]+]] = private unnamed_addr constant [3 x i8] c"bye"
// CHECK: @[[WORLD:[0-9]+]] = private unnamed_addr constant [5 x i8] c"world"
// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: [[COND:%[0-9]+]] = call i1 @main.f(%"{{.*}}String" { ptr @[[HELLO]], i64 5 })
// CHECK-NEXT: [[PREVIOUS_DEFER:%[0-9]+]] = call ptr @"{{.*}}GetThreadDefer"()
// CHECK: [[DEFER_FRAME:%[0-9]+]] = call ptr @"{{.*}}AllocU"(i64 48)
// CHECK: [[PREVIOUS_SLOT:%[0-9]+]] = getelementptr inbounds %"{{.*}}Defer", ptr [[DEFER_FRAME]], i32 0, i32 2
// CHECK-NEXT: store ptr [[PREVIOUS_DEFER]], ptr [[PREVIOUS_SLOT]]
// CHECK: [[DEFER_HEAD:%[0-9]+]] = getelementptr inbounds %"{{.*}}Defer", ptr [[DEFER_FRAME]], i32 0, i32 5
// CHECK: [[TRUE_HEAD1:%[0-9]+]] = load ptr, ptr [[DEFER_HEAD]]
// CHECK: store ptr [[TRUE_HEAD1]], ptr %{{[0-9]+}}
// CHECK: store %"{{.*}}String" { ptr @[[HELLO]], i64 5 }, ptr %{{[0-9]+}}
// CHECK: [[TRUE_NODE2:%[0-9]+]] = call ptr @"{{.*}}AllocU"(i64 32)
// CHECK: store %"{{.*}}String" { ptr @[[BYE]], i64 3 }, ptr %{{[0-9]+}}
// CHECK: [[FALSE_NODE:%[0-9]+]] = call ptr @"{{.*}}AllocU"(i64 32)
// CHECK: store %"{{.*}}String" { ptr @[[WORLD]], i64 5 }, ptr %{{[0-9]+}}
// CHECK: br i1 [[COND]], label %{{.*}}, label %{{.*}}
// CHECK-COUNT-2: call void @"{{.*}}FreeDeferNode"(ptr %{{[0-9]+}})
// CHECK: [[SAVED_DEFER:%[0-9]+]] = load %"{{.*}}Defer", ptr [[DEFER_FRAME]]
// CHECK-NEXT: [[RESTORED_DEFER:%[0-9]+]] = extractvalue %"{{.*}}Defer" [[SAVED_DEFER]], 2
// CHECK-NEXT: call void @"{{.*}}SetThreadDefer"(ptr [[RESTORED_DEFER]])
// CHECK: call void @"{{.*}}FreeDeferNode"(ptr %{{[0-9]+}})

func f(s string) bool {
	return len(s) > 2
}

func main() {
	if s := "hello"; f(s) {
		defer println(s)
	} else {
		defer println("world")
		return
	}
	defer println("bye")
}
