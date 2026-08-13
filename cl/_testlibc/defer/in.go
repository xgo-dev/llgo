// LITTEST
package main

import "github.com/goplus/lib/c"

// C varargs defers still use the Go defer chain, and each deferred printf is
// guarded by a recover frame at invocation time.
// CHECK: [[HELLO:@[0-9]+]] = private unnamed_addr constant [5 x i8] c"hello"
// CHECK: [[FORMAT:@[0-9]+]] = private unnamed_addr constant [4 x i8] c"%s\0A\00"
// CHECK: [[WORLD:@[0-9]+]] = private unnamed_addr constant [7 x i8] c"world\0A\00"
// CHECK: [[BYE:@[0-9]+]] = private unnamed_addr constant [5 x i8] c"bye\0A\00"
// CHECK-LABEL: define i1 @main.f(%"{{.*}}String" %0){{.*}} {
// CHECK: [[F_LEN:%.*]] = extractvalue %"{{.*}}String" %0, 1
// CHECK-NEXT: [[F_RESULT:%.*]] = icmp sgt i64 [[F_LEN]], 2
// CHECK-NEXT: ret i1 [[F_RESULT]]

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: [[GO_DEFER_DATA:%.*]] = call ptr @"{{.*}}GetThreadDefer"()
// CHECK: [[COND:%.*]] = call i1 @main.f(%"{{.*}}String" { ptr [[HELLO]], i64 5 })
// CHECK: [[PREV_DEFER:%.*]] = call ptr @"{{.*}}GetThreadDefer"()
// CHECK: [[FRAME:%.*]] = call ptr @"{{.*}}AllocU"(i64 48)
// CHECK: store ptr [[PREV_DEFER]], ptr {{%.*}}
// CHECK: call void @"{{.*}}SetThreadDefer"(ptr [[FRAME]])
// CHECK: [[FLAGS:%.*]] = getelementptr inbounds %"{{.*}}Defer", ptr [[FRAME]], i32 0, i32 1
// CHECK: [[HEAD:%.*]] = getelementptr inbounds %"{{.*}}Defer", ptr [[FRAME]], i32 0, i32 5
// True branch: copy the Go string to stable C storage and keep format+argument.
// CHECK: [[CSTR_BUF:%.*]] = alloca i8, i64 6
// CHECK-NEXT: [[CSTR:%.*]] = call ptr @"{{.*}}CStrCopy"(ptr [[CSTR_BUF]], %"{{.*}}String" { ptr [[HELLO]], i64 5 })
// CHECK: [[FORMAT_FLAGS0:%.*]] = load i64, ptr [[FLAGS]]
// CHECK-NEXT: [[FORMAT_FLAGS:%.*]] = or i64 [[FORMAT_FLAGS0]], 1
// CHECK-NEXT: store i64 [[FORMAT_FLAGS]], ptr [[FLAGS]]
// CHECK: [[FORMAT_PREV:%.*]] = load ptr, ptr [[HEAD]]
// CHECK: [[FORMAT_NODE:%.*]] = call ptr @"{{.*}}AllocU"(i64 32)
// CHECK: store ptr [[FORMAT_PREV]], ptr {{%.*}}
// CHECK: store ptr [[FORMAT]], ptr {{%.*}}
// CHECK: store ptr [[CSTR]], ptr {{%.*}}
// CHECK: store ptr [[FORMAT_NODE]], ptr [[HEAD]]
// Common path: bye is always the newest node.
// CHECK: [[BYE_PREV:%.*]] = load ptr, ptr [[HEAD]]
// CHECK: [[BYE_NODE:%.*]] = call ptr @"{{.*}}AllocU"(i64 24)
// CHECK: store ptr [[BYE_PREV]], ptr {{%.*}}
// CHECK: store i64 2, ptr {{%.*}}
// CHECK: store ptr [[BYE]], ptr {{%.*}}
// CHECK: store ptr [[BYE_NODE]], ptr [[HEAD]]
// False branch: world has its own flag and node.
// CHECK: [[WORLD_FLAGS0:%.*]] = load i64, ptr [[FLAGS]]
// CHECK-NEXT: [[WORLD_FLAGS:%.*]] = or i64 [[WORLD_FLAGS0]], 2
// CHECK-NEXT: store i64 [[WORLD_FLAGS]], ptr [[FLAGS]]
// CHECK: [[WORLD_PREV:%.*]] = load ptr, ptr [[HEAD]]
// CHECK: [[WORLD_NODE:%.*]] = call ptr @"{{.*}}AllocU"(i64 24)
// CHECK: store ptr [[WORLD_PREV]], ptr {{%.*}}
// CHECK: store ptr [[WORLD]], ptr {{%.*}}
// CHECK: store ptr [[WORLD_NODE]], ptr [[HEAD]]
// CHECK: br i1 [[COND]], label %{{.*}}, label %{{.*}}
// Cleanup order in the IR is bye, conditional world, then conditional %s.
// CHECK: [[FORMAT_MASK:%.*]] = and i64 {{%.*}}, 1
// CHECK: [[WORLD_MASK:%.*]] = and i64 {{%.*}}, 2
// CHECK: [[BYE_ACTIVE:%.*]] = load ptr, ptr [[HEAD]]
// CHECK-NEXT: [[BYE_VALUE:%.*]] = load { ptr, i64, ptr }, ptr [[BYE_ACTIVE]]
// CHECK: [[BYE_CALL_FORMAT:%.*]] = extractvalue { ptr, i64, ptr } [[BYE_VALUE]], 2
// CHECK-NEXT: call void @"{{.*}}FreeDeferNode"(ptr [[BYE_ACTIVE]])
// CHECK-NEXT: [[BYE_RECOVER:%.*]] = call %"{{.*}}recoverState" @"{{.*}}StartRecoverFrame"(ptr @printf)
// CHECK-NEXT: call i32 (ptr, ...) @printf(ptr [[BYE_CALL_FORMAT]])
// CHECK-NEXT: call void @"{{.*}}EndRecoverFrame"(%"{{.*}}recoverState" [[BYE_RECOVER]])
// CHECK: [[WORLD_CANDIDATE:%.*]] = load ptr, ptr [[HEAD]]
// CHECK-NEXT: icmp ne ptr [[WORLD_CANDIDATE]], null
// CHECK: [[WORLD_ACTIVE:%.*]] = load ptr, ptr [[HEAD]]
// CHECK-NEXT: [[WORLD_VALUE:%.*]] = load { ptr, i64, ptr }, ptr [[WORLD_ACTIVE]]
// CHECK: [[WORLD_CALL_FORMAT:%.*]] = extractvalue { ptr, i64, ptr } [[WORLD_VALUE]], 2
// CHECK-NEXT: call void @"{{.*}}FreeDeferNode"(ptr [[WORLD_ACTIVE]])
// CHECK-NEXT: [[WORLD_RECOVER:%.*]] = call %"{{.*}}recoverState" @"{{.*}}StartRecoverFrame"(ptr @printf)
// CHECK-NEXT: call i32 (ptr, ...) @printf(ptr [[WORLD_CALL_FORMAT]])
// CHECK-NEXT: call void @"{{.*}}EndRecoverFrame"(%"{{.*}}recoverState" [[WORLD_RECOVER]])
// CHECK: [[FORMAT_CANDIDATE:%.*]] = load ptr, ptr [[HEAD]]
// CHECK-NEXT: icmp ne ptr [[FORMAT_CANDIDATE]], null
// CHECK: [[FORMAT_ACTIVE:%.*]] = load ptr, ptr [[HEAD]]
// CHECK-NEXT: [[FORMAT_VALUE:%.*]] = load { ptr, i64, ptr, ptr }, ptr [[FORMAT_ACTIVE]]
// CHECK: [[CALL_FORMAT:%.*]] = extractvalue { ptr, i64, ptr, ptr } [[FORMAT_VALUE]], 2
// CHECK-NEXT: [[CALL_CSTR:%.*]] = extractvalue { ptr, i64, ptr, ptr } [[FORMAT_VALUE]], 3
// CHECK-NEXT: call void @"{{.*}}FreeDeferNode"(ptr [[FORMAT_ACTIVE]])
// CHECK-NEXT: [[FORMAT_RECOVER:%.*]] = call %"{{.*}}recoverState" @"{{.*}}StartRecoverFrame"(ptr @printf)
// CHECK-NEXT: call i32 (ptr, ...) @printf(ptr [[CALL_FORMAT]], ptr [[CALL_CSTR]])
// CHECK-NEXT: call void @"{{.*}}EndRecoverFrame"(%"{{.*}}recoverState" [[FORMAT_RECOVER]])

func f(s string) bool {
	return len(s) > 2
}

func main() {
	c.GoDeferData()
	if s := "hello"; f(s) {
		defer c.Printf(c.Str("%s\n"), c.AllocaCStr(s))
	} else {
		defer c.Printf(c.Str("world\n"))
	}
	defer c.Printf(c.Str("bye\n"))
}
