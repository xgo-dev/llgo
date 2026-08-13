// LITTEST
package main

import (
	"runtime"
)

// Goexit runs deferred frames without creating a recoverable panic value.
// CHECK: [[MUST_NIL:@[0-9]+]] = private unnamed_addr constant [8 x i8] c"must nil"
// CHECK: [[ERROR:@[0-9]+]] = private unnamed_addr constant [5 x i8] c"error"
// CHECK: [[MUST_ERROR:@[0-9]+]] = private unnamed_addr constant [10 x i8] c"must error"

// demo1 starts a goroutine and waits for the value sent by its ordinary defer.
// CHECK-LABEL: define void @main.demo1(){{.*}} {
// CHECK: [[D1_CH_SLOT:%.*]] = call ptr @"{{.*}}AllocZ"(i64 8)
// CHECK: [[D1_CH:%.*]] = call ptr @"{{.*}}NewChan"(i64 1, i64 0)
// CHECK: store ptr [[D1_CH]], ptr [[D1_CH_SLOT]]
// CHECK: [[D1_ENV:%.*]] = call ptr @"{{.*}}AllocU"(i64 8)
// CHECK: store ptr [[D1_CH_SLOT]], ptr {{%.*}}
// CHECK: [[D1_CLOSURE:%.*]] = insertvalue { ptr, ptr } { ptr @"main.demo1$1", ptr undef }, ptr [[D1_ENV]], 1
// CHECK: store { ptr, ptr } [[D1_CLOSURE]], ptr {{%.*}}
// CHECK: call void @"{{.*}}NewProc"(ptr @"main._llgo_routine$1", ptr {{%.*}}, i64 0)
// CHECK: [[D1_RECV_CH:%.*]] = load ptr, ptr [[D1_CH_SLOT]]
// CHECK: [[D1_RECV_BUF:%.*]] = alloca i1
// CHECK: call i1 @"{{.*}}ChanRecv"(ptr [[D1_RECV_CH]], ptr [[D1_RECV_BUF]], i64 1)
// CHECK: load i1, ptr [[D1_RECV_BUF]]

// CHECK-LABEL: define void @"main.demo1$1"(ptr {{(nest|swiftself)}} %0){{.*}} {
// CHECK: [[D1_CAPTURE:%.*]] = load { ptr }, ptr %0
// CHECK: [[D1_CH_CAPTURE:%.*]] = extractvalue { ptr } [[D1_CAPTURE]], 0
// CHECK: [[D1_DEFER_ENV:%.*]] = call ptr @"{{.*}}AllocU"(i64 8)
// CHECK: store ptr [[D1_CH_CAPTURE]], ptr {{%.*}}
// CHECK: [[D1_DEFER:%.*]] = insertvalue { ptr, ptr } { ptr @"main.demo1$1$1", ptr undef }, ptr [[D1_DEFER_ENV]], 1
// CHECK: [[D1_HEAD:%.*]] = getelementptr inbounds %"{{.*}}Defer", ptr {{%.*}}, i32 0, i32 5
// CHECK: [[D1_NODE:%.*]] = call ptr @"{{.*}}AllocU"(i64 32)
// CHECK: [[D1_NODE_FUNC:%.*]] = getelementptr inbounds { ptr, i64, { ptr, ptr } }, ptr [[D1_NODE]], i32 0, i32 2
// CHECK: store { ptr, ptr } [[D1_DEFER]], ptr [[D1_NODE_FUNC]]
// CHECK: store ptr [[D1_NODE]], ptr [[D1_HEAD]]
// CHECK-NOT: StartRecoverFrame
// CHECK: call void @runtime.Goexit()
// CHECK: [[D1_ACTIVE_NODE:%.*]] = load ptr, ptr [[D1_HEAD]]
// CHECK: [[D1_NODE_VALUE:%.*]] = load { ptr, i64, { ptr, ptr } }, ptr [[D1_ACTIVE_NODE]]
// CHECK: [[D1_DEFER_VALUE:%.*]] = extractvalue { ptr, i64, { ptr, ptr } } [[D1_NODE_VALUE]], 2
// CHECK: call void @"{{.*}}FreeDeferNode"(ptr [[D1_ACTIVE_NODE]])
// CHECK: [[D1_DEFER_CALL_ENV:%.*]] = extractvalue { ptr, ptr } [[D1_DEFER_VALUE]], 1
// CHECK: [[D1_DEFER_CALL_FN:%.*]] = extractvalue { ptr, ptr } [[D1_DEFER_VALUE]], 0
// CHECK-NOT: StartRecoverFrame
// CHECK: [[D1_DEFER_CODE:%.*]] = call ptr asm "", "=r,0"(ptr [[D1_DEFER_CALL_FN]])
// CHECK: call void [[D1_DEFER_CODE]](ptr {{(nest|swiftself)}} [[D1_DEFER_CALL_ENV]])

// CHECK-LABEL: define void @"main.demo1$1$1"(ptr {{(nest|swiftself)}} %0){{.*}} {
// CHECK: [[D1_SEND_CAPTURE:%.*]] = load { ptr }, ptr %0
// CHECK: [[D1_SEND_SLOT:%.*]] = extractvalue { ptr } [[D1_SEND_CAPTURE]], 0
// CHECK: [[D1_SEND_CH:%.*]] = load ptr, ptr [[D1_SEND_SLOT]]
// CHECK: [[D1_SEND_BUF:%.*]] = alloca i1
// CHECK: store i1 true, ptr [[D1_SEND_BUF]]
// CHECK: call i1 @"{{.*}}ChanSend"(ptr [[D1_SEND_CH]], ptr [[D1_SEND_BUF]], i64 1)

// demo2 verifies that recover in a Goexit defer observes nil, then signals completion.
// CHECK-LABEL: define void @main.demo2(){{.*}} {
// CHECK: [[D2_CH_SLOT:%.*]] = call ptr @"{{.*}}AllocZ"(i64 8)
// CHECK: [[D2_CH:%.*]] = call ptr @"{{.*}}NewChan"(i64 1, i64 0)
// CHECK: store ptr [[D2_CH]], ptr [[D2_CH_SLOT]]
// CHECK: [[D2_ENV:%.*]] = call ptr @"{{.*}}AllocU"(i64 8)
// CHECK: store ptr [[D2_CH_SLOT]], ptr {{%.*}}
// CHECK: [[D2_CLOSURE:%.*]] = insertvalue { ptr, ptr } { ptr @"main.demo2$1", ptr undef }, ptr [[D2_ENV]], 1
// CHECK: store { ptr, ptr } [[D2_CLOSURE]], ptr {{%.*}}
// CHECK: call void @"{{.*}}NewProc"(ptr @"main._llgo_routine$2", ptr {{%.*}}, i64 0)
// CHECK: [[D2_RECV_CH:%.*]] = load ptr, ptr [[D2_CH_SLOT]]
// CHECK: [[D2_RECV_BUF:%.*]] = alloca i1
// CHECK: call i1 @"{{.*}}ChanRecv"(ptr [[D2_RECV_CH]], ptr [[D2_RECV_BUF]], i64 1)
// CHECK: load i1, ptr [[D2_RECV_BUF]]

// CHECK-LABEL: define void @"main.demo2$1"(ptr {{(nest|swiftself)}} %0){{.*}} {
// CHECK: [[D2_CAPTURE:%.*]] = load { ptr }, ptr %0
// CHECK: [[D2_CH_CAPTURE:%.*]] = extractvalue { ptr } [[D2_CAPTURE]], 0
// CHECK: [[D2_DEFER_ENV:%.*]] = call ptr @"{{.*}}AllocU"(i64 8)
// CHECK: store ptr [[D2_CH_CAPTURE]], ptr {{%.*}}
// CHECK: [[D2_DEFER:%.*]] = insertvalue { ptr, ptr } { ptr @"main.demo2$1$1", ptr undef }, ptr [[D2_DEFER_ENV]], 1
// CHECK: [[D2_HEAD:%.*]] = getelementptr inbounds %"{{.*}}Defer", ptr {{%.*}}, i32 0, i32 5
// CHECK: [[D2_NODE:%.*]] = call ptr @"{{.*}}AllocU"(i64 32)
// CHECK: [[D2_NODE_FUNC:%.*]] = getelementptr inbounds { ptr, i64, { ptr, ptr } }, ptr [[D2_NODE]], i32 0, i32 2
// CHECK: store { ptr, ptr } [[D2_DEFER]], ptr [[D2_NODE_FUNC]]
// CHECK: store ptr [[D2_NODE]], ptr [[D2_HEAD]]
// CHECK: call void @runtime.Goexit()
// CHECK: [[D2_ACTIVE_NODE:%.*]] = load ptr, ptr [[D2_HEAD]]
// CHECK: [[D2_NODE_VALUE:%.*]] = load { ptr, i64, { ptr, ptr } }, ptr [[D2_ACTIVE_NODE]]
// CHECK: [[D2_DEFER_VALUE:%.*]] = extractvalue { ptr, i64, { ptr, ptr } } [[D2_NODE_VALUE]], 2
// CHECK: call void @"{{.*}}FreeDeferNode"(ptr [[D2_ACTIVE_NODE]])
// CHECK: [[D2_RECOVER_FN:%.*]] = extractvalue { ptr, ptr } [[D2_DEFER_VALUE]], 0
// CHECK: [[D2_RECOVER_STATE:%.*]] = call %"{{.*}}recoverState" @"{{.*}}StartRecoverFrame"(ptr [[D2_RECOVER_FN]])
// CHECK: [[D2_DEFER_CALL_ENV:%.*]] = extractvalue { ptr, ptr } [[D2_DEFER_VALUE]], 1
// CHECK: [[D2_DEFER_CALL_FN:%.*]] = extractvalue { ptr, ptr } [[D2_DEFER_VALUE]], 0
// CHECK: [[D2_DEFER_CODE:%.*]] = call ptr asm "", "=r,0"(ptr [[D2_DEFER_CALL_FN]])
// CHECK: call void [[D2_DEFER_CODE]](ptr {{(nest|swiftself)}} [[D2_DEFER_CALL_ENV]])
// CHECK: call void @"{{.*}}EndRecoverFrame"(%"{{.*}}recoverState" [[D2_RECOVER_STATE]])

// CHECK-LABEL: define void @"main.demo2$1$1"(ptr {{(nest|swiftself)}} %0){{.*}} {
// CHECK: [[D2_SEND_CAPTURE:%.*]] = load { ptr }, ptr %0
// CHECK: [[D2_RECOVER_TOKEN:%.*]] = alloca i8
// CHECK: call void @"{{.*}}BindRecoverFrame"(ptr @"main.demo2$1$1", ptr [[D2_RECOVER_TOKEN]])
// CHECK: [[D2_RECOVERED:%.*]] = call %"{{.*}}eface" @"{{.*}}Recover"(ptr [[D2_RECOVER_TOKEN]])
// CHECK: [[D2_IS_NIL:%.*]] = call i1 @"{{.*}}EfaceEqual"(%"{{.*}}eface" [[D2_RECOVERED]], %"{{.*}}eface" zeroinitializer)
// CHECK: [[D2_NOT_NIL:%.*]] = xor i1 [[D2_IS_NIL]], true
// CHECK: br i1 [[D2_NOT_NIL]], label %{{.*}}, label %{{.*}}
// CHECK: store %"{{.*}}String" { ptr [[MUST_NIL]], i64 8 }, ptr [[D2_MUST_NIL_BUF:%.*]], align {{[0-9]+}}
// CHECK: [[D2_MUST_NIL:%.*]] = insertvalue %"{{.*}}eface" { ptr @_llgo_string, ptr undef }, ptr [[D2_MUST_NIL_BUF]], 1
// CHECK: call void @"{{.*}}Panic"(%"{{.*}}eface" [[D2_MUST_NIL]])
// CHECK: [[D2_SEND_SLOT:%.*]] = extractvalue { ptr } [[D2_SEND_CAPTURE]], 0
// CHECK: [[D2_SEND_CH:%.*]] = load ptr, ptr [[D2_SEND_SLOT]]
// CHECK: [[D2_SEND_BUF:%.*]] = alloca i1
// CHECK: store i1 true, ptr [[D2_SEND_BUF]]
// CHECK: call i1 @"{{.*}}ChanSend"(ptr [[D2_SEND_CH]], ptr [[D2_SEND_BUF]], i64 1)

// demo3 turns Goexit's nil recover into a panic, then recovers that panic outside.
// CHECK-LABEL: define void @main.demo3(){{.*}} {
// CHECK: [[D3_CH_SLOT:%.*]] = call ptr @"{{.*}}AllocZ"(i64 8)
// CHECK: [[D3_CH:%.*]] = call ptr @"{{.*}}NewChan"(i64 1, i64 0)
// CHECK: store ptr [[D3_CH]], ptr [[D3_CH_SLOT]]
// CHECK: [[D3_ENV:%.*]] = call ptr @"{{.*}}AllocU"(i64 8)
// CHECK: store ptr [[D3_CH_SLOT]], ptr {{%.*}}
// CHECK: [[D3_CLOSURE:%.*]] = insertvalue { ptr, ptr } { ptr @"main.demo3$1", ptr undef }, ptr [[D3_ENV]], 1
// CHECK: store { ptr, ptr } [[D3_CLOSURE]], ptr {{%.*}}
// CHECK: call void @"{{.*}}NewProc"(ptr @"main._llgo_routine$3", ptr {{%.*}}, i64 0)
// CHECK: [[D3_RECV_CH:%.*]] = load ptr, ptr [[D3_CH_SLOT]]
// CHECK: [[D3_RECV_BUF:%.*]] = alloca i1
// CHECK: call i1 @"{{.*}}ChanRecv"(ptr [[D3_RECV_CH]], ptr [[D3_RECV_BUF]], i64 1)
// CHECK: load i1, ptr [[D3_RECV_BUF]]

// CHECK-LABEL: define void @"main.demo3$1"(ptr {{(nest|swiftself)}} %0){{.*}} {
// CHECK: [[D3_CAPTURE:%.*]] = load { ptr }, ptr %0
// CHECK: [[D3_CH_CAPTURE:%.*]] = extractvalue { ptr } [[D3_CAPTURE]], 0
// CHECK: [[D3_OUTER_ENV:%.*]] = call ptr @"{{.*}}AllocU"(i64 8)
// CHECK: store ptr [[D3_CH_CAPTURE]], ptr {{%.*}}
// CHECK: [[D3_OUTER_DEFER:%.*]] = insertvalue { ptr, ptr } { ptr @"main.demo3$1$1", ptr undef }, ptr [[D3_OUTER_ENV]], 1
// CHECK: [[D3_HEAD:%.*]] = getelementptr inbounds %"{{.*}}Defer", ptr {{%.*}}, i32 0, i32 5
// CHECK: [[D3_INNER_STATE:%.*]] = call %"{{.*}}recoverState" @"{{.*}}StartRecoverFrame"(ptr @"main.demo3$1$2")
// CHECK-NEXT: call void @"main.demo3$1$2"()
// CHECK-NEXT: call void @"{{.*}}EndRecoverFrame"(%"{{.*}}recoverState" [[D3_INNER_STATE]])
// CHECK: [[D3_OUTER_NODE:%.*]] = call ptr @"{{.*}}AllocU"(i64 32)
// CHECK: [[D3_OUTER_NODE_FUNC:%.*]] = getelementptr inbounds { ptr, i64, { ptr, ptr } }, ptr [[D3_OUTER_NODE]], i32 0, i32 2
// CHECK: store { ptr, ptr } [[D3_OUTER_DEFER]], ptr [[D3_OUTER_NODE_FUNC]]
// CHECK: store ptr [[D3_OUTER_NODE]], ptr [[D3_HEAD]]
// CHECK: call void @runtime.Goexit()
// CHECK: [[D3_HAS_NODE:%.*]] = load ptr, ptr [[D3_HEAD]]
// CHECK: icmp ne ptr [[D3_HAS_NODE]], null
// CHECK: [[D3_ACTIVE_NODE:%.*]] = load ptr, ptr [[D3_HEAD]]
// CHECK: [[D3_NODE_VALUE:%.*]] = load { ptr, i64, { ptr, ptr } }, ptr [[D3_ACTIVE_NODE]]
// CHECK: [[D3_OUTER_VALUE:%.*]] = extractvalue { ptr, i64, { ptr, ptr } } [[D3_NODE_VALUE]], 2
// CHECK: call void @"{{.*}}FreeDeferNode"(ptr [[D3_ACTIVE_NODE]])
// CHECK: [[D3_OUTER_FN:%.*]] = extractvalue { ptr, ptr } [[D3_OUTER_VALUE]], 0
// CHECK: [[D3_OUTER_STATE:%.*]] = call %"{{.*}}recoverState" @"{{.*}}StartRecoverFrame"(ptr [[D3_OUTER_FN]])
// CHECK: [[D3_OUTER_CALL_ENV:%.*]] = extractvalue { ptr, ptr } [[D3_OUTER_VALUE]], 1
// CHECK: [[D3_OUTER_CALL_FN:%.*]] = extractvalue { ptr, ptr } [[D3_OUTER_VALUE]], 0
// CHECK: [[D3_OUTER_CODE:%.*]] = call ptr asm "", "=r,0"(ptr [[D3_OUTER_CALL_FN]])
// CHECK: call void [[D3_OUTER_CODE]](ptr {{(nest|swiftself)}} [[D3_OUTER_CALL_ENV]])
// CHECK: call void @"{{.*}}EndRecoverFrame"(%"{{.*}}recoverState" [[D3_OUTER_STATE]])

// CHECK-LABEL: define void @"main.demo3$1$1"(ptr {{(nest|swiftself)}} %0){{.*}} {
// CHECK: [[D3_OUTER_CAPTURE:%.*]] = load { ptr }, ptr %0
// CHECK: [[D3_OUTER_TOKEN:%.*]] = alloca i8
// CHECK: call void @"{{.*}}BindRecoverFrame"(ptr @"main.demo3$1$1", ptr [[D3_OUTER_TOKEN]])
// CHECK: [[D3_RECOVERED_ERROR:%.*]] = call %"{{.*}}eface" @"{{.*}}Recover"(ptr [[D3_OUTER_TOKEN]])
// CHECK: store %"{{.*}}String" { ptr [[ERROR]], i64 5 }, ptr [[D3_ERROR_BUF:%.*]], align {{[0-9]+}}
// CHECK: [[D3_ERROR_EFACE:%.*]] = insertvalue %"{{.*}}eface" { ptr @_llgo_string, ptr undef }, ptr [[D3_ERROR_BUF]], 1
// CHECK: [[D3_IS_ERROR:%.*]] = call i1 @"{{.*}}EfaceEqual"(%"{{.*}}eface" [[D3_RECOVERED_ERROR]], %"{{.*}}eface" [[D3_ERROR_EFACE]])
// CHECK: [[D3_NOT_ERROR:%.*]] = xor i1 [[D3_IS_ERROR]], true
// CHECK: br i1 [[D3_NOT_ERROR]], label %{{.*}}, label %{{.*}}
// CHECK: store %"{{.*}}String" { ptr [[MUST_ERROR]], i64 10 }, ptr [[D3_MUST_ERROR_BUF:%.*]], align {{[0-9]+}}
// CHECK: [[D3_MUST_ERROR:%.*]] = insertvalue %"{{.*}}eface" { ptr @_llgo_string, ptr undef }, ptr [[D3_MUST_ERROR_BUF]], 1
// CHECK: call void @"{{.*}}Panic"(%"{{.*}}eface" [[D3_MUST_ERROR]])
// CHECK: [[D3_SEND_SLOT:%.*]] = extractvalue { ptr } [[D3_OUTER_CAPTURE]], 0
// CHECK: [[D3_SEND_CH:%.*]] = load ptr, ptr [[D3_SEND_SLOT]]
// CHECK: [[D3_SEND_BUF:%.*]] = alloca i1
// CHECK: store i1 true, ptr [[D3_SEND_BUF]]
// CHECK: call i1 @"{{.*}}ChanSend"(ptr [[D3_SEND_CH]], ptr [[D3_SEND_BUF]], i64 1)

// CHECK-LABEL: define void @"main.demo3$1$2"(){{.*}} {
// CHECK: [[D3_INNER_TOKEN:%.*]] = alloca i8
// CHECK: call void @"{{.*}}BindRecoverFrame"(ptr @"main.demo3$1$2", ptr [[D3_INNER_TOKEN]])
// CHECK: [[D3_RECOVERED_NIL:%.*]] = call %"{{.*}}eface" @"{{.*}}Recover"(ptr [[D3_INNER_TOKEN]])
// CHECK: [[D3_INNER_IS_NIL:%.*]] = call i1 @"{{.*}}EfaceEqual"(%"{{.*}}eface" [[D3_RECOVERED_NIL]], %"{{.*}}eface" zeroinitializer)
// CHECK: [[D3_INNER_NOT_NIL:%.*]] = xor i1 [[D3_INNER_IS_NIL]], true
// CHECK: br i1 [[D3_INNER_NOT_NIL]], label %{{.*}}, label %{{.*}}
// CHECK: store %"{{.*}}String" { ptr [[MUST_NIL]], i64 8 }, ptr [[D3_INNER_MUST_NIL_BUF:%.*]], align {{[0-9]+}}
// CHECK: [[D3_INNER_MUST_NIL:%.*]] = insertvalue %"{{.*}}eface" { ptr @_llgo_string, ptr undef }, ptr [[D3_INNER_MUST_NIL_BUF]], 1
// CHECK: call void @"{{.*}}Panic"(%"{{.*}}eface" [[D3_INNER_MUST_NIL]])
// CHECK: store %"{{.*}}String" { ptr [[ERROR]], i64 5 }, ptr [[D3_INNER_ERROR_BUF:%.*]], align {{[0-9]+}}
// CHECK: [[D3_INNER_ERROR:%.*]] = insertvalue %"{{.*}}eface" { ptr @_llgo_string, ptr undef }, ptr [[D3_INNER_ERROR_BUF]], 1
// CHECK: call void @"{{.*}}Panic"(%"{{.*}}eface" [[D3_INNER_ERROR]])

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT: call void @main.demo1()
// CHECK-NEXT: call void @main.demo2()
// CHECK-NEXT: call void @main.demo3()
// CHECK-NEXT: ret void

func main() {
	demo1()
	demo2()
	demo3()
}

func demo1() {
	ch := make(chan bool)
	go func() {
		defer func() {
			ch <- true
		}()
		runtime.Goexit()
	}()
	<-ch
}

func demo2() {
	ch := make(chan bool)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				panic("must nil")
			}
			ch <- true
		}()
		runtime.Goexit()
	}()
	<-ch
}

func demo3() {
	ch := make(chan bool)
	go func() {
		defer func() {
			r := recover()
			if r != "error" {
				panic("must error")
			}
			ch <- true
		}()
		defer func() {
			if r := recover(); r != nil {
				panic("must nil")
			}
			panic("error")
		}()
		runtime.Goexit()
	}()
	<-ch
}
