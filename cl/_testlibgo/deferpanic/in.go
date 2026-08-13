// LITTEST
package main

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: [[DEFER_HEAD:%[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.GetThreadDefer"()
// CHECK: call void @"{{.*}}/runtime/internal/runtime.SetThreadDefer"(ptr [[DEFER_RECORD:%[0-9]+]])
// CHECK: [[PANIC_PREV:%[0-9]+]] = load ptr, ptr [[PANIC_LIST_SLOT:%[0-9]+]]
// CHECK: [[PANIC_NODE_ALLOC:%[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 32)
// CHECK: store %"{{.*}}eface" [[PANIC_VALUE:%[0-9]+]], ptr %{{[0-9]+}}
// CHECK-NEXT: store ptr [[PANIC_NODE_ALLOC]], ptr [[PANIC_LIST_SLOT]]
// CHECK: [[RECOVER_STATE:%[0-9]+]] = call %"{{.*}}recoverState" @"{{.*}}/runtime/internal/runtime.StartRecoverFrame"(ptr @"main.main$1")
// CHECK-NEXT: call void @"main.main$1"()
// CHECK-NEXT: call void @"{{.*}}/runtime/internal/runtime.EndRecoverFrame"(%"{{.*}}recoverState" [[RECOVER_STATE]])
// CHECK: [[PANIC_NODE:%[0-9]+]] = load ptr, ptr [[PANIC_LIST_SLOT]]
// CHECK-NEXT: [[PANIC_RECORD:%[0-9]+]] = load { ptr, i64, %"{{.*}}eface" }, ptr [[PANIC_NODE]]
// CHECK: [[DEFERRED_PANIC:%[0-9]+]] = extractvalue { ptr, i64, %"{{.*}}eface" } [[PANIC_RECORD]], 2
// CHECK-NEXT: call void @"{{.*}}/runtime/internal/runtime.FreeDeferNode"(ptr [[PANIC_NODE]])
// CHECK-NEXT: call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}eface" [[DEFERRED_PANIC]])
// CHECK-NEXT: unreachable
// CHECK-LABEL: define void @"main.main$1"(){{.*}} {
// CHECK: call void @"{{.*}}/runtime/internal/runtime.BindRecoverFrame"(ptr @"main.main$1", ptr [[RECOVER_TOKEN:%[0-9]+]])
// CHECK-NEXT: [[RECOVERED:%[0-9]+]] = call %"{{.*}}eface" @"{{.*}}/runtime/internal/runtime.Recover"(ptr [[RECOVER_TOKEN]])
// CHECK-NEXT: [[RECOVERED_TYPE:%[0-9]+]] = extractvalue %"{{.*}}eface" [[RECOVERED]], 0
// CHECK-NEXT: [[RECOVERED_IS_STRING:%[0-9]+]] = icmp eq ptr [[RECOVERED_TYPE]], @_llgo_string
// CHECK: [[RECOVERED_DATA:%[0-9]+]] = extractvalue %"{{.*}}eface" [[RECOVERED]], 1
// CHECK-NEXT: [[RECOVERED_STRING:%[0-9]+]] = load %"{{.*}}String", ptr [[RECOVERED_DATA]]
// CHECK-NEXT: call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}String" [[RECOVERED_STRING]])
func main() {
	defer func() {
		e := recover()
		println(e.(string))
	}()
	defer panic("panic in defer")
	println("run main")
}
