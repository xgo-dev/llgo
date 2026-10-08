// LITTEST
// Scope: common
package main

// This package owns two distinct loop lowerings without another package
// compile: dynamic defer nodes and allocation-free constant strings.
// CHECK: [[LOOP_TEXT:@[0-9]+]] = private unnamed_addr constant [4 x i8] c"loop"
// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: [[PREV_DEFER:%.*]] = call ptr @"{{.*}}GetThreadDefer"()
// CHECK: [[FRAME:%.*]] = call noalias dereferenceable(48) ptr @"{{.*}}AllocU"(i64 48)
// CHECK: store ptr [[PREV_DEFER]], ptr {{%.*}}
// CHECK: call void @"{{.*}}SetThreadDefer"(ptr [[FRAME]])
// CHECK: [[HEAD:%.*]] = getelementptr inbounds nuw %"{{.*}}Defer", ptr [[FRAME]], i32 0, i32 5
// CHECK: [[I:%.*]] = phi i64 [ 0, %{{.*}} ], [ [[NEXT_I:%.*]], %{{.*}} ]
// CHECK: [[IN_RANGE:%.*]] = icmp slt i64 [[I]], 3
// CHECK: br i1 [[IN_RANGE]], label %{{.*}}, label %{{.*}}
// CHECK: [[OLD_HEAD:%.*]] = load ptr, ptr [[HEAD]]
// CHECK: [[NODE:%.*]] = call noalias dereferenceable(40) ptr @"{{.*}}AllocU"(i64 40)
// CHECK: store ptr [[OLD_HEAD]], ptr %{{.*}}
// CHECK: store %"{{.*}}String" { ptr [[LOOP_TEXT]], i64 4 }, ptr %{{.*}}
// CHECK: store i64 [[I]], ptr %{{.*}}
// CHECK: store ptr [[NODE]], ptr [[HEAD]]
// CHECK: [[NEXT_I]] = add i64 [[I]], 1
// CHECK: call i64 @main.zStringLoop()
// CHECK: [[HEAD_CANDIDATE:%.*]] = load ptr, ptr [[HEAD]]
// CHECK-NEXT: [[HAS_NODE:%.*]] = icmp ne ptr [[HEAD_CANDIDATE]], null
// CHECK: [[NODE_HEADER:%.*]] = load { ptr, i64 }, ptr [[HEAD_CANDIDATE]]
// CHECK-NEXT: [[DEFER_KIND:%.*]] = extractvalue { ptr, i64 } [[NODE_HEADER]], 1
// CHECK: icmp eq i64 [[DEFER_KIND]], 0
// CHECK: [[DUE_NODE:%.*]] = load ptr, ptr [[HEAD]]
// CHECK-NEXT: [[HAS_DUE_NODE:%.*]] = icmp ne ptr [[DUE_NODE]], null
// CHECK: [[ACTIVE_NODE:%.*]] = load ptr, ptr [[HEAD]]
// CHECK: [[NODE_VALUE:%.*]] = load { ptr, i64, %"{{.*}}String", i64 }, ptr [[ACTIVE_NODE]]
// CHECK: [[NEXT_NODE:%.*]] = extractvalue { ptr, i64, %"{{.*}}String", i64 } [[NODE_VALUE]], 0
// CHECK: store ptr [[NEXT_NODE]], ptr [[HEAD]]
// CHECK: [[DEFER_TEXT:%.*]] = extractvalue { ptr, i64, %"{{.*}}String", i64 } [[NODE_VALUE]], 2
// CHECK: [[DEFER_I:%.*]] = extractvalue { ptr, i64, %"{{.*}}String", i64 } [[NODE_VALUE]], 3
// CHECK: call void @"{{.*}}FreeDeferNode"(ptr [[ACTIVE_NODE]])
// CHECK: call void @"{{.*}}PrintString"(%"{{.*}}String" [[DEFER_TEXT]])
// CHECK: call void @"{{.*}}PrintInt"(i64 [[DEFER_I]])

// CHECK-LABEL: define i64 @main.zStringLoop(){{.*}} {
// CHECK-NOT: Alloc
// CHECK: call i64 @main.zStringLen(%"{{.*}}String" { ptr @{{[0-9]+}}, i64 5 })
// CHECK-NOT: Alloc
// CHECK: ret i64

func main() {
	for i := 0; i < 3; i++ {
		defer println("loop", i)
	}
	if zStringLoop() != 20 {
		panic("constant string loop")
	}
}
