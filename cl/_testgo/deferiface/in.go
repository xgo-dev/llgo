// LITTEST
package main

type resetter interface {
	Reset()
}

type item struct {
	value int
}

func (p *item) Reset() {
	println("reset", p.value)
}

func run(v resetter) {
	defer v.Reset()
	println("body")
}

func main() {
	run(&item{42})
}

// CHECK-LABEL: define void @"main.(*item).Reset"(ptr %0){{.*}} {
// CHECK: [[RESET_VALUE_PTR:%.*]] = getelementptr inbounds %main.item, ptr %0, i32 0, i32 0
// CHECK-NEXT: [[RESET_VALUE:%.*]] = load i64, ptr [[RESET_VALUE_PTR]]
// CHECK: call void @"{{.*}}PrintInt"(i64 [[RESET_VALUE]])

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: [[ITEM:%.*]] = call ptr @"{{.*}}AllocZ"(i64 8)
// CHECK-NEXT: [[ITEM_VALUE:%.*]] = getelementptr inbounds %main.item, ptr [[ITEM]], i32 0, i32 0
// CHECK-NEXT: store i64 42, ptr [[ITEM_VALUE]]
// CHECK: [[ITEM_ITAB:%.*]] = call ptr @"{{.*}}NewItab"(ptr @"_llgo_iface${{[-A-Za-z0-9_]+}}", ptr @"*_llgo_main.item")
// CHECK-NEXT: [[ITEM_IFACE0:%.*]] = insertvalue %"{{.*}}iface" undef, ptr [[ITEM_ITAB]], 0
// CHECK-NEXT: [[ITEM_IFACE:%.*]] = insertvalue %"{{.*}}iface" [[ITEM_IFACE0]], ptr [[ITEM]], 1
// CHECK-NEXT: call void @main.run(%"{{.*}}iface" [[ITEM_IFACE]])

// run resolves Reset once, stores its code+receiver pair as the defer payload,
// then invokes exactly that payload from cleanup under a recover frame.
// CHECK-LABEL: define void @main.run(%"{{.*}}iface" %0){{.*}} {
// CHECK: [[RESET_RECEIVER:%.*]] = call ptr @"{{.*}}IfacePtrData"(%"{{.*}}iface" %0)
// CHECK-NEXT: [[RESET_ITAB:%.*]] = extractvalue %"{{.*}}iface" %0, 0
// CHECK-NEXT: [[RESET_SLOT:%.*]] = getelementptr ptr, ptr [[RESET_ITAB]], i64 3
// CHECK-NEXT: [[RESET_METHOD:%.*]] = load ptr, ptr [[RESET_SLOT]]
// CHECK-NEXT: [[RESET_CALL0:%.*]] = insertvalue { ptr, ptr } undef, ptr [[RESET_METHOD]], 0
// CHECK-NEXT: [[RESET_CALL:%.*]] = insertvalue { ptr, ptr } [[RESET_CALL0]], ptr [[RESET_RECEIVER]], 1
// CHECK: [[PREV_DEFER:%.*]] = call ptr @"{{.*}}GetThreadDefer"()
// CHECK: [[FRAME:%.*]] = call ptr @"{{.*}}AllocU"(i64 48)
// CHECK: store ptr [[PREV_DEFER]], ptr {{%.*}}
// CHECK: call void @"{{.*}}SetThreadDefer"(ptr [[FRAME]])
// CHECK: [[HEAD:%.*]] = getelementptr inbounds %"{{.*}}Defer", ptr [[FRAME]], i32 0, i32 5
// CHECK: [[HEAD_CANDIDATE:%.*]] = load ptr, ptr [[HEAD]]
// CHECK-NEXT: icmp ne ptr [[HEAD_CANDIDATE]], null
// CHECK: call void @"{{.*}}Rethrow"(ptr [[PREV_DEFER]])
// CHECK: [[OLD_HEAD:%.*]] = load ptr, ptr [[HEAD]]
// CHECK: [[NODE:%.*]] = call ptr @"{{.*}}AllocU"(i64 32)
// CHECK: [[NODE_PREV:%.*]] = getelementptr inbounds { ptr, i64, { ptr, ptr } }, ptr [[NODE]], i32 0, i32 0
// CHECK-NEXT: store ptr [[OLD_HEAD]], ptr [[NODE_PREV]]
// CHECK: [[NODE_CALL:%.*]] = getelementptr inbounds { ptr, i64, { ptr, ptr } }, ptr [[NODE]], i32 0, i32 2
// CHECK-NEXT: store { ptr, ptr } [[RESET_CALL]], ptr [[NODE_CALL]]
// CHECK-NEXT: store ptr [[NODE]], ptr [[HEAD]]
// CHECK: call void @"{{.*}}PrintString"(%"{{.*}}String" { ptr @{{[0-9]+}}, i64 4 })
// CHECK: [[ACTIVE_NODE:%.*]] = load ptr, ptr [[HEAD]]
// CHECK-NEXT: [[NODE_VALUE:%.*]] = load { ptr, i64, { ptr, ptr } }, ptr [[ACTIVE_NODE]]
// CHECK: [[NEXT_NODE:%.*]] = extractvalue { ptr, i64, { ptr, ptr } } [[NODE_VALUE]], 0
// CHECK-NEXT: store ptr [[NEXT_NODE]], ptr [[HEAD]]
// CHECK-NEXT: [[DEFERRED_RESET:%.*]] = extractvalue { ptr, i64, { ptr, ptr } } [[NODE_VALUE]], 2
// CHECK-NEXT: call void @"{{.*}}FreeDeferNode"(ptr [[ACTIVE_NODE]])
// CHECK-NEXT: [[DEFERRED_RESET_FN:%.*]] = extractvalue { ptr, ptr } [[DEFERRED_RESET]], 0
// CHECK-NEXT: [[RESET_RECOVER:%.*]] = call %"{{.*}}recoverState" @"{{.*}}StartRecoverFrame"(ptr [[DEFERRED_RESET_FN]])
// CHECK-NEXT: [[DEFERRED_RESET_RECEIVER:%.*]] = extractvalue { ptr, ptr } [[DEFERRED_RESET]], 1
// CHECK-NEXT: [[DEFERRED_RESET_CODE:%.*]] = extractvalue { ptr, ptr } [[DEFERRED_RESET]], 0
// CHECK-NEXT: call void [[DEFERRED_RESET_CODE]](ptr [[DEFERRED_RESET_RECEIVER]])
// CHECK-NEXT: call void @"{{.*}}EndRecoverFrame"(%"{{.*}}recoverState" [[RESET_RECOVER]])
