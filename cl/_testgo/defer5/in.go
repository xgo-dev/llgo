// LITTEST
package main

// Bind the four messages so the checks below prove which defer/panic payload
// flows through each lowering path without depending on numbered globals.
// CHECK-DAG: [[DEFER_A:@[0-9]+]] = private unnamed_addr constant [1 x i8] c"A"
// CHECK-DAG: [[DEFER_B:@[0-9]+]] = private unnamed_addr constant [1 x i8] c"B"
// CHECK-DAG: [[PANIC_MAIN:@[0-9]+]] = private unnamed_addr constant [13 x i8] c"panic in main"
// CHECK-DAG: [[PRINT_DEFER1:@[0-9]+]] = private unnamed_addr constant [10 x i8] c"in defer 1"
// CHECK-DAG: [[PANIC_DEFER1:@[0-9]+]] = private unnamed_addr constant [16 x i8] c"panic in defer 1"
// CHECK-DAG: [[PRINT_DEFER2:@[0-9]+]] = private unnamed_addr constant [10 x i8] c"in defer 2"
// CHECK-DAG: [[PANIC_DEFER2:@[0-9]+]] = private unnamed_addr constant [16 x i8] c"panic in defer 2"

// CHECK-LABEL: define void @main.main(){{.*}} {
func main() {
	// The function installs one defer frame and preserves both the previous
	// thread frame and the initial resume block used after longjmp.
	// CHECK: [[OUTER_DEFER:%[0-9]+]] = call ptr @"{{.*}}.GetThreadDefer"()
	// CHECK-NEXT: [[DEFER_JMPBUF:%[0-9]+]] = alloca i8, i64 {{[0-9]+}}, align 1
	// CHECK-NEXT: [[DEFER_FRAME:%[0-9]+]] = call ptr @"{{.*}}.AllocU"(i64 48)
	// CHECK: [[FRAME_BUF_FIELD:%[0-9]+]] = getelementptr inbounds %"{{.*}}.Defer", ptr [[DEFER_FRAME]], i32 0, i32 0
	// CHECK-NEXT: store ptr [[DEFER_JMPBUF]], ptr [[FRAME_BUF_FIELD]]
	// CHECK: [[FRAME_PREV_FIELD:%[0-9]+]] = getelementptr inbounds %"{{.*}}.Defer", ptr [[DEFER_FRAME]], i32 0, i32 2
	// CHECK-NEXT: store ptr [[OUTER_DEFER]], ptr [[FRAME_PREV_FIELD]]
	// CHECK: [[FRAME_RESUME_FIELD:%[0-9]+]] = getelementptr inbounds %"{{.*}}.Defer", ptr [[DEFER_FRAME]], i32 0, i32 3
	// CHECK-NEXT: store ptr blockaddress(@main.main, %[[OUTER_RESUME:[A-Za-z0-9_]+]]), ptr [[FRAME_RESUME_FIELD]]
	// CHECK-NEXT: call void @"{{.*}}.SetThreadDefer"(ptr [[DEFER_FRAME]])
	// CHECK: [[DEFER_STATE_FIELD:%[0-9]+]] = getelementptr inbounds %"{{.*}}.Defer", ptr [[DEFER_FRAME]], i32 0, i32 1
	// CHECK-NEXT: [[DEFER_RESUME_FIELD:%[0-9]+]] = getelementptr inbounds %"{{.*}}.Defer", ptr [[DEFER_FRAME]], i32 0, i32 3
	// CHECK-NEXT: [[DEFER_RETHROW_FIELD:%[0-9]+]] = getelementptr inbounds %"{{.*}}.Defer", ptr [[DEFER_FRAME]], i32 0, i32 4
	// CHECK-NEXT: [[DEFER_HEAD_FIELD:%[0-9]+]] = getelementptr inbounds %"{{.*}}.Defer", ptr [[DEFER_FRAME]], i32 0, i32 5
	// CHECK: [[SETJMP_RESULT:%[0-9]+]] = call i32 @{{(__)?sigsetjmp}}(ptr [[DEFER_JMPBUF]], i32 0)
	// CHECK-NEXT: [[FIRST_ENTRY:%[0-9]+]] = icmp eq i32 [[SETJMP_RESULT]], 0
	// CHECK-NEXT: br i1 [[FIRST_ENTRY]], label %{{.*}}, label %{{.*}}

	// CHECK: [[OUTER_RESUME]]:
	// CHECK-NEXT: store ptr blockaddress(@main.main, %[[DEFER2_BLOCK:[A-Za-z0-9_]+]]), ptr [[DEFER_RESUME_FIELD]]
	// CHECK: call void @"{{.*}}.Rethrow"(ptr [[OUTER_DEFER]])

	// Plain println defers are registered as linked nodes.  Their state and
	// payload identify A as the outer defer and B as the inner one.
	// CHECK: [[DEFER_A_NODE:%[0-9]+]] = call ptr @"{{.*}}.AllocU"(i64 32)
	// CHECK: [[DEFER_A_STATE:%[0-9]+]] = getelementptr inbounds { ptr, i64, %"{{.*}}.String" }, ptr [[DEFER_A_NODE]], i32 0, i32 1
	// CHECK-NEXT: store i64 0, ptr [[DEFER_A_STATE]]
	// CHECK-NEXT: [[DEFER_A_ARG:%[0-9]+]] = getelementptr inbounds { ptr, i64, %"{{.*}}.String" }, ptr [[DEFER_A_NODE]], i32 0, i32 2
	// CHECK-NEXT: store %"{{.*}}.String" { ptr [[DEFER_A]], i64 1 }, ptr [[DEFER_A_ARG]]
	// CHECK-NEXT: store ptr [[DEFER_A_NODE]], ptr [[DEFER_HEAD_FIELD]]
	// CHECK: [[DEFER_B_NODE:%[0-9]+]] = call ptr @"{{.*}}.AllocU"(i64 32)
	// CHECK: [[DEFER_B_STATE:%[0-9]+]] = getelementptr inbounds { ptr, i64, %"{{.*}}.String" }, ptr [[DEFER_B_NODE]], i32 0, i32 1
	// CHECK-NEXT: store i64 3, ptr [[DEFER_B_STATE]]
	// CHECK-NEXT: [[DEFER_B_ARG:%[0-9]+]] = getelementptr inbounds { ptr, i64, %"{{.*}}.String" }, ptr [[DEFER_B_NODE]], i32 0, i32 2
	// CHECK-NEXT: store %"{{.*}}.String" { ptr [[DEFER_B]], i64 1 }, ptr [[DEFER_B_ARG]]
	// CHECK-NEXT: store ptr [[DEFER_B_NODE]], ptr [[DEFER_HEAD_FIELD]]
	// CHECK: [[MAIN_PANIC_BOX:%[0-9]+]] = call ptr @"{{.*}}.AllocU"(i64 16)
	// CHECK-NEXT: store %"{{.*}}.String" { ptr [[PANIC_MAIN]], i64 13 }, ptr [[MAIN_PANIC_BOX]]
	// CHECK-NEXT: [[MAIN_PANIC_VALUE:%[0-9]+]] = insertvalue %"{{.*}}.eface" { ptr @_llgo_string, ptr undef }, ptr [[MAIN_PANIC_BOX]], 1
	// CHECK-NEXT: call void @"{{.*}}.Panic"(%"{{.*}}.eface" [[MAIN_PANIC_VALUE]])

	// The state machine invokes defer 2 first, then enters a recover frame for
	// defer 1.  Capturing block labels keeps the relation without pinning their
	// generated numbers.
	// CHECK: [[LANDING_BLOCK:[A-Za-z0-9_]+]]:
	// CHECK-NEXT: store ptr blockaddress(@main.main, %[[RETHROW_BLOCK:[A-Za-z0-9_]+]]), ptr [[DEFER_RETHROW_FIELD]]
	// CHECK: indirectbr ptr %{{[0-9]+}}, [{{.*}}label %[[DEFER2_BLOCK]]{{.*}}]
	// CHECK: [[AFTER_RECOVER:[A-Za-z0-9_]+]]:
	// CHECK-NEXT: store ptr blockaddress(@main.main, %[[RETHROW_BLOCK]]), ptr [[DEFER_RESUME_FIELD]]
	// CHECK: [[RECOVER_BLOCK:[A-Za-z0-9_]+]]:
	// CHECK-NEXT: store ptr blockaddress(@main.main, %[[AFTER_RECOVER]]), ptr [[DEFER_RESUME_FIELD]]
	// CHECK: [[RECOVER_STATE:%[0-9]+]] = call %"{{.*}}.recoverState" @"{{.*}}.StartRecoverFrame"(ptr @"main.main$1")
	// CHECK-NEXT: call void @"main.main$1"()
	// CHECK-NEXT: call void @"{{.*}}.EndRecoverFrame"(%"{{.*}}.recoverState" [[RECOVER_STATE]])
	// CHECK: [[DEFER2_BLOCK]]:
	// CHECK-NEXT: store ptr blockaddress(@main.main, %[[RECOVER_BLOCK]]), ptr [[DEFER_RESUME_FIELD]]
	// CHECK: call void @"main.main$2"()
	// CHECK-NEXT: br label %[[RECOVER_BLOCK]]

	// Both linked println nodes are popped, freed, and printed through the
	// values read from the same defer-head field.
	// CHECK: [[POP1:%[0-9]+]] = load ptr, ptr [[DEFER_HEAD_FIELD]]
	// CHECK-NEXT: [[POP1_RECORD:%[0-9]+]] = load { ptr, i64, %"{{.*}}.String" }, ptr [[POP1]]
	// CHECK-NEXT: [[POP1_PREV:%[0-9]+]] = extractvalue { ptr, i64, %"{{.*}}.String" } [[POP1_RECORD]], 0
	// CHECK-NEXT: store ptr [[POP1_PREV]], ptr [[DEFER_HEAD_FIELD]]
	// CHECK-NEXT: [[POP1_TEXT:%[0-9]+]] = extractvalue { ptr, i64, %"{{.*}}.String" } [[POP1_RECORD]], 2
	// CHECK-NEXT: call void @"{{.*}}.FreeDeferNode"(ptr [[POP1]])
	// CHECK-NEXT: call void @"{{.*}}.PrintString"(%"{{.*}}.String" [[POP1_TEXT]])
	// CHECK: [[POP2:%[0-9]+]] = load ptr, ptr [[DEFER_HEAD_FIELD]]
	// CHECK-NEXT: [[POP2_RECORD:%[0-9]+]] = load { ptr, i64, %"{{.*}}.String" }, ptr [[POP2]]
	// CHECK-NEXT: [[POP2_PREV:%[0-9]+]] = extractvalue { ptr, i64, %"{{.*}}.String" } [[POP2_RECORD]], 0
	// CHECK-NEXT: store ptr [[POP2_PREV]], ptr [[DEFER_HEAD_FIELD]]
	// CHECK-NEXT: [[POP2_TEXT:%[0-9]+]] = extractvalue { ptr, i64, %"{{.*}}.String" } [[POP2_RECORD]], 2
	// CHECK-NEXT: call void @"{{.*}}.FreeDeferNode"(ptr [[POP2]])
	// CHECK-NEXT: call void @"{{.*}}.PrintString"(%"{{.*}}.String" [[POP2_TEXT]])
	// CHECK: [[DEFER_FRAME_VALUE:%[0-9]+]] = load %"{{.*}}.Defer", ptr [[DEFER_FRAME]]
	// CHECK-NEXT: [[PREVIOUS_DEFER:%[0-9]+]] = extractvalue %"{{.*}}.Defer" [[DEFER_FRAME_VALUE]], 2
	// CHECK-NEXT: call void @"{{.*}}.SetThreadDefer"(ptr [[PREVIOUS_DEFER]])
	defer println("A")
	defer func() {
		if e := recover(); e != nil {
			println("in defer 1")
			panic("panic in defer 1")
		}
	}()
	defer func() {
		println("in defer 2")
		panic("panic in defer 2")
	}()
	defer println("B")
	panic("panic in main")
}

// CHECK-LABEL: define void @"main.main$1"(){{.*}} {
// CHECK: [[RECOVER_TOKEN:%[0-9]+]] = alloca i8
// CHECK-NEXT: call void @"{{.*}}.BindRecoverFrame"(ptr @"main.main$1", ptr [[RECOVER_TOKEN]])
// CHECK-NEXT: [[RECOVERED:%[0-9]+]] = call %"{{.*}}.eface" @"{{.*}}.Recover"(ptr [[RECOVER_TOKEN]])
// CHECK-NEXT: [[RECOVER_EMPTY:%[0-9]+]] = call i1 @"{{.*}}.EfaceEqual"(%"{{.*}}.eface" [[RECOVERED]], %"{{.*}}.eface" zeroinitializer)
// CHECK-NEXT: [[RECOVER_NONEMPTY:%[0-9]+]] = xor i1 [[RECOVER_EMPTY]], true
// CHECK-NEXT: br i1 [[RECOVER_NONEMPTY]], label %{{.*}}, label %{{.*}}
// CHECK: call void @"{{.*}}.PrintString"(%"{{.*}}.String" { ptr [[PRINT_DEFER1]], i64 10 })
// CHECK: [[DEFER1_PANIC_BOX:%[0-9]+]] = call ptr @"{{.*}}.AllocU"(i64 16)
// CHECK-NEXT: store %"{{.*}}.String" { ptr [[PANIC_DEFER1]], i64 16 }, ptr [[DEFER1_PANIC_BOX]]
// CHECK-NEXT: [[DEFER1_PANIC:%[0-9]+]] = insertvalue %"{{.*}}.eface" { ptr @_llgo_string, ptr undef }, ptr [[DEFER1_PANIC_BOX]], 1
// CHECK-NEXT: call void @"{{.*}}.Panic"(%"{{.*}}.eface" [[DEFER1_PANIC]])
// CHECK-LABEL: define void @"main.main$2"(){{.*}} {
// CHECK: call void @"{{.*}}.PrintString"(%"{{.*}}.String" { ptr [[PRINT_DEFER2]], i64 10 })
// CHECK: [[DEFER2_PANIC_BOX:%[0-9]+]] = call ptr @"{{.*}}.AllocU"(i64 16)
// CHECK-NEXT: store %"{{.*}}.String" { ptr [[PANIC_DEFER2]], i64 16 }, ptr [[DEFER2_PANIC_BOX]]
// CHECK-NEXT: [[DEFER2_PANIC:%[0-9]+]] = insertvalue %"{{.*}}.eface" { ptr @_llgo_string, ptr undef }, ptr [[DEFER2_PANIC_BOX]], 1
// CHECK-NEXT: call void @"{{.*}}.Panic"(%"{{.*}}.eface" [[DEFER2_PANIC]])
