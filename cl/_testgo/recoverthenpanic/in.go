// LITTEST
package main

// Recovering and immediately deferring a new panic must bind the active frame,
// enqueue the recovered value, and rethrow through the surrounding frame.
// CHECK: [[WILL_PANIC:@[0-9]+]] = private unnamed_addr constant [19 x i8] c"will panic in defer"
// CHECK: [[END:@[0-9]+]] = private unnamed_addr constant [3 x i8] c"end"
// CHECK: [[MAIN_PANIC:@[0-9]+]] = private unnamed_addr constant [13 x i8] c"panic in main"
// CHECK-LABEL: define void @main.End(){{.*}} {
// CHECK: [[RECOVER_TOKEN:%.*]] = alloca i8
// CHECK-NEXT: call void @"{{.*}}BindRecoverFrame"(ptr @main.End, ptr [[RECOVER_TOKEN]])
// CHECK-NEXT: [[RECOVERED:%.*]] = call %"{{.*}}eface" @"{{.*}}Recover"(ptr [[RECOVER_TOKEN]])
// CHECK-NEXT: [[RECOVER_IS_NIL:%.*]] = call i1 @"{{.*}}EfaceEqual"(%"{{.*}}eface" [[RECOVERED]], %"{{.*}}eface" zeroinitializer)
// CHECK-NEXT: [[RECOVER_NONNIL:%.*]] = xor i1 [[RECOVER_IS_NIL]], true
// CHECK: [[PREV_DEFER:%.*]] = call ptr @"{{.*}}GetThreadDefer"()
// CHECK: [[FRAME:%.*]] = call ptr @"{{.*}}AllocU"(i64 48)
// CHECK: store ptr [[PREV_DEFER]], ptr {{%.*}}
// CHECK: call void @"{{.*}}SetThreadDefer"(ptr [[FRAME]])
// CHECK: [[FLAGS:%.*]] = getelementptr inbounds %"{{.*}}Defer", ptr [[FRAME]], i32 0, i32 1
// CHECK: [[HEAD:%.*]] = getelementptr inbounds %"{{.*}}Defer", ptr [[FRAME]], i32 0, i32 5
// On a non-nil recovery, defer panic(recovered) captures that exact eface.
// CHECK: [[FLAGS0:%.*]] = load i64, ptr [[FLAGS]]
// CHECK-NEXT: [[FLAGS1:%.*]] = or i64 [[FLAGS0]], 1
// CHECK-NEXT: store i64 [[FLAGS1]], ptr [[FLAGS]]
// CHECK: [[OLD_HEAD:%.*]] = load ptr, ptr [[HEAD]]
// CHECK: [[NODE:%.*]] = call ptr @"{{.*}}AllocU"(i64 32)
// CHECK: [[NODE_PREV:%.*]] = getelementptr inbounds { ptr, i64, %"{{.*}}eface" }, ptr [[NODE]], i32 0, i32 0
// CHECK-NEXT: store ptr [[OLD_HEAD]], ptr [[NODE_PREV]]
// CHECK: [[NODE_RECOVERED:%.*]] = getelementptr inbounds { ptr, i64, %"{{.*}}eface" }, ptr [[NODE]], i32 0, i32 2
// CHECK-NEXT: store %"{{.*}}eface" [[RECOVERED]], ptr [[NODE_RECOVERED]]
// CHECK-NEXT: store ptr [[NODE]], ptr [[HEAD]]
// CHECK: call void @"{{.*}}PrintString"(%"{{.*}}String" { ptr [[WILL_PANIC]], i64 19 })
// CHECK: call void @"{{.*}}PrintString"(%"{{.*}}String" { ptr [[END]], i64 3 })
// CHECK: br i1 [[RECOVER_NONNIL]], label %{{.*}}, label %{{.*}}
// CHECK: [[DEFER_MASK:%.*]] = and i64 {{%.*}}, 1
// CHECK: call void @"{{.*}}Rethrow"(ptr [[PREV_DEFER]])
// CHECK: [[NODE_CANDIDATE:%.*]] = load ptr, ptr [[HEAD]]
// CHECK-NEXT: icmp ne ptr [[NODE_CANDIDATE]], null
// CHECK: [[ACTIVE_NODE:%.*]] = load ptr, ptr [[HEAD]]
// CHECK-NEXT: [[NODE_VALUE:%.*]] = load { ptr, i64, %"{{.*}}eface" }, ptr [[ACTIVE_NODE]]
// CHECK: [[NEXT_NODE:%.*]] = extractvalue { ptr, i64, %"{{.*}}eface" } [[NODE_VALUE]], 0
// CHECK-NEXT: store ptr [[NEXT_NODE]], ptr [[HEAD]]
// CHECK-NEXT: [[REPANIC:%.*]] = extractvalue { ptr, i64, %"{{.*}}eface" } [[NODE_VALUE]], 2
// CHECK-NEXT: call void @"{{.*}}FreeDeferNode"(ptr [[ACTIVE_NODE]])
// CHECK-NEXT: call void @"{{.*}}Panic"(%"{{.*}}eface" [[REPANIC]])
// CHECK-NEXT: unreachable

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: [[MAIN_PREV_DEFER:%.*]] = call ptr @"{{.*}}GetThreadDefer"()
// CHECK: [[MAIN_FRAME:%.*]] = call ptr @"{{.*}}AllocU"(i64 48)
// CHECK: store ptr [[MAIN_PREV_DEFER]], ptr {{%.*}}
// CHECK: call void @"{{.*}}SetThreadDefer"(ptr [[MAIN_FRAME]])
// The deferred End call is wrapped in the recovery frame that lets End recover.
// CHECK: [[END_RECOVER:%.*]] = call %"{{.*}}recoverState" @"{{.*}}StartRecoverFrame"(ptr @main.End)
// CHECK-NEXT: call void @main.End()
// CHECK-NEXT: call void @"{{.*}}EndRecoverFrame"(%"{{.*}}recoverState" [[END_RECOVER]])
// CHECK: call void @"{{.*}}Rethrow"(ptr [[MAIN_PREV_DEFER]])
// The original panic value entering that cleanup is the source string.
// CHECK: [[MAIN_PANIC_BUF:%.*]] = call ptr @"{{.*}}AllocU"(i64 16)
// CHECK-NEXT: store %"{{.*}}String" { ptr [[MAIN_PANIC]], i64 13 }, ptr [[MAIN_PANIC_BUF]]
// CHECK-NEXT: [[MAIN_PANIC_EFACE:%.*]] = insertvalue %"{{.*}}eface" { ptr @_llgo_string, ptr undef }, ptr [[MAIN_PANIC_BUF]], 1
// CHECK-NEXT: call void @"{{.*}}Panic"(%"{{.*}}eface" [[MAIN_PANIC_EFACE]])
// CHECK-NEXT: unreachable

func End() {
	if recovered := recover(); recovered != nil {
		// Record but don't stop the panic.
		defer panic(recovered)
		println("will panic in defer")
	}
	println("end")
}

func main() {
	defer End()
	panic("panic in main")
}
