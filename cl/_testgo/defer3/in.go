// LITTEST
package main

// A panic in fail must execute fail's own defer, restore the enclosing defer
// chain, then let main's selected and outer defers run while rethrowing.
// CHECK: @[[D3_BYE:[0-9]+]] = private unnamed_addr constant [3 x i8] c"bye"
// CHECK: @[[D3_PANIC:[0-9]+]] = private unnamed_addr constant [13 x i8] c"panic message"
// CHECK: @[[D3_HELLO:[0-9]+]] = private unnamed_addr constant [5 x i8] c"hello"
// CHECK: @[[D3_WORLD:[0-9]+]] = private unnamed_addr constant [5 x i8] c"world"
// CHECK-LABEL: define void @main.fail(){{.*}} {
// CHECK: [[D3_FAIL_PREV:%[0-9]+]] = call ptr @"{{.*}}GetThreadDefer"()
// CHECK: [[D3_FAIL_FRAME:%[0-9]+]] = call ptr @"{{.*}}AllocU"(i64 48)
// CHECK: [[D3_FAIL_PREV_SLOT:%[0-9]+]] = getelementptr inbounds %"{{.*}}Defer", ptr [[D3_FAIL_FRAME]], i32 0, i32 2
// CHECK-NEXT: store ptr [[D3_FAIL_PREV]], ptr [[D3_FAIL_PREV_SLOT]]
// CHECK: call void @"{{.*}}SetThreadDefer"(ptr [[D3_FAIL_FRAME]])
// CHECK: [[D3_FAIL_HEAD:%[0-9]+]] = getelementptr inbounds %"{{.*}}Defer", ptr [[D3_FAIL_FRAME]], i32 0, i32 5
// CHECK: call void @"{{.*}}Rethrow"(ptr [[D3_FAIL_PREV]])
// CHECK: [[D3_FAIL_OLD_HEAD:%[0-9]+]] = load ptr, ptr [[D3_FAIL_HEAD]]
// CHECK-NEXT: [[D3_FAIL_NODE:%[0-9]+]] = call ptr @"{{.*}}AllocU"(i64 32)
// CHECK: store ptr [[D3_FAIL_OLD_HEAD]], ptr %{{[0-9]+}}
// CHECK: store %"{{.*}}String" { ptr @[[D3_BYE]], i64 3 }, ptr %{{[0-9]+}}
// CHECK-NEXT: store ptr [[D3_FAIL_NODE]], ptr [[D3_FAIL_HEAD]]
// CHECK: store %"{{.*}}String" { ptr @[[D3_PANIC]], i64 13 }, ptr [[D3_PANIC_BOX:%[0-9]+]]
// CHECK-NEXT: [[D3_PANIC_EFACE:%[0-9]+]] = insertvalue %"{{.*}}eface" { ptr @_llgo_string, ptr undef }, ptr [[D3_PANIC_BOX]], 1
// CHECK-NEXT: call void @"{{.*}}Panic"(%"{{.*}}eface" [[D3_PANIC_EFACE]])
// CHECK: [[D3_RUN_NODE:%[0-9]+]] = load ptr, ptr [[D3_FAIL_HEAD]]
// CHECK: [[D3_RUN_PAYLOAD:%[0-9]+]] = extractvalue { ptr, i64, %"{{.*}}String" } %{{[0-9]+}}, 2
// CHECK-NEXT: call void @"{{.*}}FreeDeferNode"(ptr [[D3_RUN_NODE]])
// CHECK-NEXT: call void @"{{.*}}PrintString"(%"{{.*}}String" [[D3_RUN_PAYLOAD]])
// CHECK: [[D3_FAIL_SAVED:%[0-9]+]] = load %"{{.*}}Defer", ptr [[D3_FAIL_FRAME]]
// CHECK-NEXT: [[D3_FAIL_RESTORE:%[0-9]+]] = extractvalue %"{{.*}}Defer" [[D3_FAIL_SAVED]], 2
// CHECK-NEXT: call void @"{{.*}}SetThreadDefer"(ptr [[D3_FAIL_RESTORE]])
// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: [[D3_MAIN_PREV:%[0-9]+]] = call ptr @"{{.*}}GetThreadDefer"()
// CHECK: [[D3_MAIN_FRAME:%[0-9]+]] = call ptr @"{{.*}}AllocU"(i64 48)
// CHECK: store ptr [[D3_MAIN_PREV]], ptr %{{[0-9]+}}
// CHECK: call void @"{{.*}}SetThreadDefer"(ptr [[D3_MAIN_FRAME]])
// CHECK: store %"{{.*}}String" { ptr @[[D3_HELLO]], i64 5 }, ptr %{{[0-9]+}}
// CHECK: call void @main.fail()
// CHECK: store %"{{.*}}String" { ptr @[[D3_WORLD]], i64 5 }, ptr %{{[0-9]+}}
// CHECK: call void @"{{.*}}Rethrow"(ptr [[D3_MAIN_PREV]])
// CHECK: [[D3_COND:%[0-9]+]] = call i1 @main.f(%"{{.*}}String" { ptr @[[D3_HELLO]], i64 5 })
// CHECK-NEXT: br i1 [[D3_COND]], label %{{.*}}, label %{{.*}}
// CHECK: call void @"main.main$1"()
// CHECK: [[D3_MAIN_SAVED:%[0-9]+]] = load %"{{.*}}Defer", ptr [[D3_MAIN_FRAME]]
// CHECK-NEXT: [[D3_MAIN_RESTORE:%[0-9]+]] = extractvalue %"{{.*}}Defer" [[D3_MAIN_SAVED]], 2
// CHECK-NEXT: call void @"{{.*}}SetThreadDefer"(ptr [[D3_MAIN_RESTORE]])
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
	println("unreachable")
}
