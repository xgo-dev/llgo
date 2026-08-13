// LITTEST
package main

// The defer must survive the loop/control-flow joins and be released on the
// shared exit path.
// CHECK: [[BYE:@[0-9]+]] = private unnamed_addr constant [3 x i8] c"bye"
// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: [[PREV_DEFER:%.*]] = call ptr @"{{.*}}GetThreadDefer"()
// CHECK: [[FRAME:%.*]] = call ptr @"{{.*}}AllocU"(i64 48)
// CHECK: store ptr [[PREV_DEFER]], ptr {{%.*}}
// CHECK: call void @"{{.*}}SetThreadDefer"(ptr [[FRAME]])
// CHECK: [[FLAGS:%.*]] = getelementptr inbounds %"{{.*}}Defer", ptr [[FRAME]], i32 0, i32 1
// CHECK: [[HEAD:%.*]] = getelementptr inbounds %"{{.*}}Defer", ptr [[FRAME]], i32 0, i32 5
// The first empty range still forms a valid loop and reaches defer registration.
// CHECK: [[FIRST_I:%.*]] = phi i64 [ -1, %{{.*}} ], [ [[FIRST_NEXT:%.*]], %{{.*}} ]
// CHECK-NEXT: [[FIRST_NEXT]] = add i64 [[FIRST_I]], 1
// CHECK-NEXT: [[FIRST_MORE:%.*]] = icmp slt i64 [[FIRST_NEXT]], 0
// CHECK-NEXT: br i1 [[FIRST_MORE]], label %{{.*}}, label %{{.*}}
// CHECK: [[OLD_FLAGS:%.*]] = load i64, ptr [[FLAGS]]
// CHECK-NEXT: [[DEFER_FLAGS:%.*]] = or i64 [[OLD_FLAGS]], 1
// CHECK-NEXT: store i64 [[DEFER_FLAGS]], ptr [[FLAGS]]
// CHECK: [[OLD_HEAD:%.*]] = load ptr, ptr [[HEAD]]
// CHECK: [[NODE:%.*]] = call ptr @"{{.*}}AllocU"(i64 32)
// CHECK: [[NODE_PREV:%.*]] = getelementptr inbounds { ptr, i64, %"{{.*}}String" }, ptr [[NODE]], i32 0, i32 0
// CHECK-NEXT: store ptr [[OLD_HEAD]], ptr [[NODE_PREV]]
// CHECK: [[NODE_TEXT:%.*]] = getelementptr inbounds { ptr, i64, %"{{.*}}String" }, ptr [[NODE]], i32 0, i32 2
// CHECK-NEXT: store %"{{.*}}String" { ptr [[BYE]], i64 3 }, ptr [[NODE_TEXT]]
// CHECK-NEXT: store ptr [[NODE]], ptr [[HEAD]]
// The second empty range exits through the shared cleanup instead of bypassing it.
// CHECK: [[SECOND_I:%.*]] = phi i64 [ -1, %{{.*}} ], [ [[SECOND_NEXT:%.*]], %{{.*}} ]
// CHECK-NEXT: [[SECOND_NEXT]] = add i64 [[SECOND_I]], 1
// CHECK-NEXT: [[SECOND_MORE:%.*]] = icmp slt i64 [[SECOND_NEXT]], 0
// CHECK-NEXT: br i1 [[SECOND_MORE]], label %{{.*}}, label %{{.*}}
// CHECK: [[EXIT_FLAGS:%.*]] = load i64, ptr [[FLAGS]]
// CHECK-NEXT: [[DEFER_BIT:%.*]] = and i64 [[EXIT_FLAGS]], 1
// CHECK-NEXT: [[HAS_DEFER:%.*]] = icmp ne i64 [[DEFER_BIT]], 0
// CHECK: [[NODE_CANDIDATE:%.*]] = load ptr, ptr [[HEAD]]
// CHECK-NEXT: icmp ne ptr [[NODE_CANDIDATE]], null
// CHECK: [[ACTIVE_NODE:%.*]] = load ptr, ptr [[HEAD]]
// CHECK: [[NODE_VALUE:%.*]] = load { ptr, i64, %"{{.*}}String" }, ptr [[ACTIVE_NODE]]
// CHECK: [[NEXT_NODE:%.*]] = extractvalue { ptr, i64, %"{{.*}}String" } [[NODE_VALUE]], 0
// CHECK-NEXT: store ptr [[NEXT_NODE]], ptr [[HEAD]]
// CHECK: [[DEFER_TEXT:%.*]] = extractvalue { ptr, i64, %"{{.*}}String" } [[NODE_VALUE]], 2
// CHECK-NEXT: call void @"{{.*}}FreeDeferNode"(ptr [[ACTIVE_NODE]])
// CHECK-NEXT: call void @"{{.*}}PrintString"(%"{{.*}}String" [[DEFER_TEXT]])

func main() {
	syms := []int{}
	for range syms {
	}
	defer println("bye")
	for range syms {
	}
}
