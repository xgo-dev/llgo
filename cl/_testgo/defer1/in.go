// LITTEST
package main

// CHECK: [[HELLO:@[0-9]+]] = private unnamed_addr constant [5 x i8] c"hello"
// CHECK: [[BYE:@[0-9]+]] = private unnamed_addr constant [3 x i8] c"bye"
// CHECK: [[WORLD:@[0-9]+]] = private unnamed_addr constant [5 x i8] c"world"
// CHECK: [[HI:@[0-9]+]] = private unnamed_addr constant [2 x i8] c"hi"
// CHECK-LABEL: define i1 @main.f(%"{{.*}}String" %0){{.*}} {
// CHECK: [[F_LEN:%.*]] = extractvalue %"{{.*}}String" %0, 1
// CHECK-NEXT: [[F_RESULT:%.*]] = icmp sgt i64 [[F_LEN]], 2
// CHECK-NEXT: ret i1 [[F_RESULT]]

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: [[PREV_DEFER:%.*]] = call ptr @"{{.*}}GetThreadDefer"()
// CHECK: [[FRAME:%.*]] = call ptr @"{{.*}}AllocU"(i64 48)
// CHECK: store ptr [[PREV_DEFER]], ptr {{%.*}}
// CHECK: call void @"{{.*}}SetThreadDefer"(ptr [[FRAME]])
// CHECK: [[FLAGS:%.*]] = getelementptr inbounds %"{{.*}}Defer", ptr [[FRAME]], i32 0, i32 1
// CHECK: [[HEAD:%.*]] = getelementptr inbounds %"{{.*}}Defer", ptr [[FRAME]], i32 0, i32 5
// True branch: capture s, then bye, as distinct nodes and flag bits.
// CHECK: [[HELLO_FLAGS0:%.*]] = load i64, ptr [[FLAGS]]
// CHECK-NEXT: [[HELLO_FLAGS:%.*]] = or i64 [[HELLO_FLAGS0]], 1
// CHECK-NEXT: store i64 [[HELLO_FLAGS]], ptr [[FLAGS]]
// CHECK: [[HELLO_PREV:%.*]] = load ptr, ptr [[HEAD]]
// CHECK: [[HELLO_NODE:%.*]] = call ptr @"{{.*}}AllocU"(i64 32)
// CHECK: store ptr [[HELLO_PREV]], ptr {{%.*}}
// CHECK: store i64 1, ptr {{%.*}}
// CHECK: store %"{{.*}}String" { ptr [[HELLO]], i64 5 }, ptr {{%.*}}
// CHECK: store ptr [[HELLO_NODE]], ptr [[HEAD]]
// CHECK: [[BYE_FLAGS0:%.*]] = load i64, ptr [[FLAGS]]
// CHECK-NEXT: [[BYE_FLAGS:%.*]] = or i64 [[BYE_FLAGS0]], 2
// CHECK-NEXT: store i64 [[BYE_FLAGS]], ptr [[FLAGS]]
// CHECK: [[BYE_PREV:%.*]] = load ptr, ptr [[HEAD]]
// CHECK: [[BYE_NODE:%.*]] = call ptr @"{{.*}}AllocU"(i64 32)
// CHECK: store ptr [[BYE_PREV]], ptr {{%.*}}
// CHECK: store i64 2, ptr {{%.*}}
// CHECK: store %"{{.*}}String" { ptr [[BYE]], i64 3 }, ptr {{%.*}}
// CHECK: store ptr [[BYE_NODE]], ptr [[HEAD]]
// False branch: capture world and mark its own cleanup bit.
// CHECK: [[WORLD_FLAGS0:%.*]] = load i64, ptr [[FLAGS]]
// CHECK-NEXT: [[WORLD_FLAGS:%.*]] = or i64 [[WORLD_FLAGS0]], 4
// CHECK-NEXT: store i64 [[WORLD_FLAGS]], ptr [[FLAGS]]
// CHECK: [[WORLD_PREV:%.*]] = load ptr, ptr [[HEAD]]
// CHECK: [[WORLD_NODE:%.*]] = call ptr @"{{.*}}AllocU"(i64 32)
// CHECK: store ptr [[WORLD_PREV]], ptr {{%.*}}
// CHECK: store i64 3, ptr {{%.*}}
// CHECK: store %"{{.*}}String" { ptr [[WORLD]], i64 5 }, ptr {{%.*}}
// CHECK: store ptr [[WORLD_NODE]], ptr [[HEAD]]
// The shared cleanup first tests the false-branch bit.
// CHECK: [[WORLD_MASK:%.*]] = and i64 {{%.*}}, 4
// CHECK: [[COND:%.*]] = call i1 @main.f(%"{{.*}}String" { ptr [[HELLO]], i64 5 })
// CHECK-NEXT: br i1 [[COND]], label %{{.*}}, label %{{.*}}
// The direct closure is the final defer after the string-node cleanups.
// CHECK: call void @"main.main$1"()
// The true path tests bye before hello, matching LIFO registration order.
// CHECK: [[HELLO_MASK:%.*]] = and i64 {{%.*}}, 1
// CHECK: [[BYE_MASK:%.*]] = and i64 {{%.*}}, 2
// Each active bit pops and prints the node currently at the head.
// CHECK: [[WORLD_CANDIDATE:%.*]] = load ptr, ptr [[HEAD]]
// CHECK-NEXT: icmp ne ptr [[WORLD_CANDIDATE]], null
// CHECK: [[WORLD_ACTIVE:%.*]] = load ptr, ptr [[HEAD]]
// CHECK: [[WORLD_VALUE:%.*]] = load { ptr, i64, %"{{.*}}String" }, ptr [[WORLD_ACTIVE]]
// CHECK: [[WORLD_NEXT:%.*]] = extractvalue { ptr, i64, %"{{.*}}String" } [[WORLD_VALUE]], 0
// CHECK-NEXT: store ptr [[WORLD_NEXT]], ptr [[HEAD]]
// CHECK: [[WORLD_TEXT:%.*]] = extractvalue { ptr, i64, %"{{.*}}String" } [[WORLD_VALUE]], 2
// CHECK-NEXT: call void @"{{.*}}FreeDeferNode"(ptr [[WORLD_ACTIVE]])
// CHECK-NEXT: call void @"{{.*}}PrintString"(%"{{.*}}String" [[WORLD_TEXT]])
// CHECK: [[BYE_CANDIDATE:%.*]] = load ptr, ptr [[HEAD]]
// CHECK-NEXT: icmp ne ptr [[BYE_CANDIDATE]], null
// CHECK: [[BYE_ACTIVE:%.*]] = load ptr, ptr [[HEAD]]
// CHECK: [[BYE_VALUE:%.*]] = load { ptr, i64, %"{{.*}}String" }, ptr [[BYE_ACTIVE]]
// CHECK: [[BYE_NEXT:%.*]] = extractvalue { ptr, i64, %"{{.*}}String" } [[BYE_VALUE]], 0
// CHECK-NEXT: store ptr [[BYE_NEXT]], ptr [[HEAD]]
// CHECK: [[BYE_TEXT:%.*]] = extractvalue { ptr, i64, %"{{.*}}String" } [[BYE_VALUE]], 2
// CHECK-NEXT: call void @"{{.*}}FreeDeferNode"(ptr [[BYE_ACTIVE]])
// CHECK-NEXT: call void @"{{.*}}PrintString"(%"{{.*}}String" [[BYE_TEXT]])
// CHECK: [[HELLO_CANDIDATE:%.*]] = load ptr, ptr [[HEAD]]
// CHECK-NEXT: icmp ne ptr [[HELLO_CANDIDATE]], null
// CHECK: [[HELLO_ACTIVE:%.*]] = load ptr, ptr [[HEAD]]
// CHECK: [[HELLO_VALUE:%.*]] = load { ptr, i64, %"{{.*}}String" }, ptr [[HELLO_ACTIVE]]
// CHECK: [[HELLO_NEXT:%.*]] = extractvalue { ptr, i64, %"{{.*}}String" } [[HELLO_VALUE]], 0
// CHECK-NEXT: store ptr [[HELLO_NEXT]], ptr [[HEAD]]
// CHECK: [[HELLO_TEXT:%.*]] = extractvalue { ptr, i64, %"{{.*}}String" } [[HELLO_VALUE]], 2
// CHECK-NEXT: call void @"{{.*}}FreeDeferNode"(ptr [[HELLO_ACTIVE]])
// CHECK-NEXT: call void @"{{.*}}PrintString"(%"{{.*}}String" [[HELLO_TEXT]])
// CHECK-LABEL: define void @"main.main$1"(){{.*}} {
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}String" { ptr [[HI]], i64 2 })
// CHECK-NEXT: call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT: ret void

func f(s string) bool {
	return len(s) > 2
}

func main() {
	defer func() {
		println("hi")
	}()
	if s := "hello"; f(s) {
		defer println(s)
	} else {
		defer println("world")
		return
	}
	defer println("bye")
}
