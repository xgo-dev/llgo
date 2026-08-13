// LITTEST
package main

// fail's recovery defer is associated with its own recover frame; after it
// consumes the string panic, fail's ordinary defer runs and main continues.
// CHECK: @[[D4_BYE:[0-9]+]] = private unnamed_addr constant [3 x i8] c"bye"
// CHECK: @[[D4_PANIC:[0-9]+]] = private unnamed_addr constant [13 x i8] c"panic message"
// CHECK: @[[D4_HELLO:[0-9]+]] = private unnamed_addr constant [5 x i8] c"hello"
// CHECK: @[[D4_WORLD:[0-9]+]] = private unnamed_addr constant [5 x i8] c"world"
// CHECK-LABEL: define void @main.fail(){{.*}} {
// CHECK: [[D4_FAIL_PREV:%[0-9]+]] = call ptr @"{{.*}}GetThreadDefer"()
// CHECK: [[D4_FAIL_FRAME:%[0-9]+]] = call ptr @"{{.*}}AllocU"(i64 48)
// CHECK: store ptr [[D4_FAIL_PREV]], ptr %{{[0-9]+}}
// CHECK: call void @"{{.*}}SetThreadDefer"(ptr [[D4_FAIL_FRAME]])
// CHECK: [[D4_FAIL_HEAD:%[0-9]+]] = getelementptr inbounds %"{{.*}}Defer", ptr [[D4_FAIL_FRAME]], i32 0, i32 5
// CHECK: [[D4_RECOVER_STATE:%[0-9]+]] = call %"{{.*}}recoverState" @"{{.*}}StartRecoverFrame"(ptr @"main.fail$1")
// CHECK-NEXT: call void @"main.fail$1"()
// CHECK-NEXT: call void @"{{.*}}EndRecoverFrame"(%"{{.*}}recoverState" [[D4_RECOVER_STATE]])
// CHECK: call void @"{{.*}}Rethrow"(ptr [[D4_FAIL_PREV]])
// CHECK: [[D4_FAIL_OLD_HEAD:%[0-9]+]] = load ptr, ptr [[D4_FAIL_HEAD]]
// CHECK-NEXT: [[D4_FAIL_NODE:%[0-9]+]] = call ptr @"{{.*}}AllocU"(i64 32)
// CHECK: store ptr [[D4_FAIL_OLD_HEAD]], ptr %{{[0-9]+}}
// CHECK: store %"{{.*}}String" { ptr @[[D4_BYE]], i64 3 }, ptr %{{[0-9]+}}
// CHECK-NEXT: store ptr [[D4_FAIL_NODE]], ptr [[D4_FAIL_HEAD]]
// CHECK: store %"{{.*}}String" { ptr @[[D4_PANIC]], i64 13 }, ptr [[D4_PANIC_BOX:%[0-9]+]]
// CHECK-NEXT: [[D4_PANIC_EFACE:%[0-9]+]] = insertvalue %"{{.*}}eface" { ptr @_llgo_string, ptr undef }, ptr [[D4_PANIC_BOX]], 1
// CHECK-NEXT: call void @"{{.*}}Panic"(%"{{.*}}eface" [[D4_PANIC_EFACE]])
// CHECK: [[D4_PENDING_NODE:%[0-9]+]] = load ptr, ptr [[D4_FAIL_HEAD]]
// CHECK-NEXT: [[D4_HAS_NODE:%[0-9]+]] = icmp ne ptr [[D4_PENDING_NODE]], null
// CHECK-NEXT: br i1 [[D4_HAS_NODE]], label %{{.*}}, label %{{.*}}
// CHECK: [[D4_RUN_NODE:%[0-9]+]] = load ptr, ptr [[D4_FAIL_HEAD]]
// CHECK: [[D4_RUN_PAYLOAD:%[0-9]+]] = extractvalue { ptr, i64, %"{{.*}}String" } %{{[0-9]+}}, 2
// CHECK-NEXT: call void @"{{.*}}FreeDeferNode"(ptr [[D4_RUN_NODE]])
// CHECK-NEXT: call void @"{{.*}}PrintString"(%"{{.*}}String" [[D4_RUN_PAYLOAD]])
// CHECK: [[D4_FAIL_SAVED:%[0-9]+]] = load %"{{.*}}Defer", ptr [[D4_FAIL_FRAME]]
// CHECK-NEXT: [[D4_FAIL_RESTORE:%[0-9]+]] = extractvalue %"{{.*}}Defer" [[D4_FAIL_SAVED]], 2
// CHECK-NEXT: call void @"{{.*}}SetThreadDefer"(ptr [[D4_FAIL_RESTORE]])
// CHECK-LABEL: define void @"main.fail$1"(){{.*}} {
// CHECK: call void @"{{.*}}BindRecoverFrame"(ptr @"main.fail$1", ptr [[D4_RECOVER_TOKEN:%[0-9]+]])
// CHECK-NEXT: [[D4_RECOVERED:%[0-9]+]] = call %"{{.*}}eface" @"{{.*}}Recover"(ptr [[D4_RECOVER_TOKEN]])
// CHECK-NEXT: [[D4_IS_NIL:%[0-9]+]] = call i1 @"{{.*}}EfaceEqual"(%"{{.*}}eface" [[D4_RECOVERED]], %"{{.*}}eface" zeroinitializer)
// CHECK-NEXT: [[D4_HAS_PANIC:%[0-9]+]] = xor i1 [[D4_IS_NIL]], true
// CHECK-NEXT: br i1 [[D4_HAS_PANIC]], label %{{.*}}, label %{{.*}}
// CHECK: [[D4_RECOVER_TYPE:%[0-9]+]] = extractvalue %"{{.*}}eface" [[D4_RECOVERED]], 0
// CHECK-NEXT: [[D4_IS_STRING:%[0-9]+]] = icmp eq ptr [[D4_RECOVER_TYPE]], @_llgo_string
// CHECK-NEXT: br i1 [[D4_IS_STRING]], label %{{.*}}, label %{{.*}}
// CHECK: [[D4_RECOVER_DATA:%[0-9]+]] = extractvalue %"{{.*}}eface" [[D4_RECOVERED]], 1
// CHECK-NEXT: [[D4_RECOVER_STRING:%[0-9]+]] = load %"{{.*}}String", ptr [[D4_RECOVER_DATA]]
// CHECK: call void @"{{.*}}PrintString"(%"{{.*}}String" [[D4_RECOVER_STRING]])
// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: [[D4_MAIN_PREV:%[0-9]+]] = call ptr @"{{.*}}GetThreadDefer"()
// CHECK: [[D4_MAIN_FRAME:%[0-9]+]] = call ptr @"{{.*}}AllocU"(i64 48)
// CHECK: store ptr [[D4_MAIN_PREV]], ptr %{{[0-9]+}}
// CHECK: call void @"{{.*}}SetThreadDefer"(ptr [[D4_MAIN_FRAME]])
// CHECK: store %"{{.*}}String" { ptr @[[D4_HELLO]], i64 5 }, ptr %{{[0-9]+}}
// CHECK: call void @main.fail()
// CHECK-NEXT: call void @"{{.*}}PrintString"(%"{{.*}}String" { ptr @{{[0-9]+}}, i64 9 })
// CHECK: store %"{{.*}}String" { ptr @[[D4_WORLD]], i64 5 }, ptr %{{[0-9]+}}
// CHECK: [[D4_COND:%[0-9]+]] = call i1 @main.f(%"{{.*}}String" { ptr @[[D4_HELLO]], i64 5 })
// CHECK-NEXT: br i1 [[D4_COND]], label %{{.*}}, label %{{.*}}
// CHECK: call void @"main.main$1"()
// CHECK: [[D4_MAIN_SAVED:%[0-9]+]] = load %"{{.*}}Defer", ptr [[D4_MAIN_FRAME]]
// CHECK-NEXT: [[D4_MAIN_RESTORE:%[0-9]+]] = extractvalue %"{{.*}}Defer" [[D4_MAIN_SAVED]], 2
// CHECK-NEXT: call void @"{{.*}}SetThreadDefer"(ptr [[D4_MAIN_RESTORE]])
// CHECK-COUNT-2: call void @"{{.*}}FreeDeferNode"(ptr %{{[0-9]+}})
// CHECK-LABEL: define void @"main.main$1"(){{.*}} {
// CHECK: call void @"{{.*}}PrintString"(%"{{.*}}String" { ptr @{{[0-9]+}}, i64 2 })
// CHECK-NEXT: call void @"{{.*}}PrintByte"(i8 10)
// CHECK-NEXT: ret void

func f(s string) bool {
	return len(s) > 2
}

func fail() {
	defer println("bye")
	defer func() {
		if e := recover(); e != nil {
			println("recover:", e.(string))
		}
	}()
	panic("panic message")
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
	fail()
	println("reachable")
}
