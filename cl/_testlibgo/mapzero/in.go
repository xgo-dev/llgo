// LITTEST
package main

import "fmt"

var a = 0

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: [[RECOVER_STATE:%.*]] = call %"{{.*}}recoverState" @"{{.*}}/runtime/internal/runtime.StartRecoverFrame"(ptr @"main.main$1")
// CHECK: call void @"main.main$1"()
// CHECK: call void @"{{.*}}/runtime/internal/runtime.EndRecoverFrame"(%"{{.*}}recoverState" [[RECOVER_STATE]])
// The zero-length array check uses a as the index; the lowered nil map is still explicit.
// CHECK: [[INDEX:%.*]] = load i64, ptr @main.a
// CHECK: [[INDEX_NEG:%.*]] = icmp slt i64 [[INDEX]], 0
// CHECK: [[INDEX_OOB:%.*]] = icmp uge i64 [[INDEX]], 0
// CHECK: [[INDEX_BAD:%.*]] = or i1 [[INDEX_OOB]], [[INDEX_NEG]]
// CHECK: call void @"{{.*}}/runtime/internal/runtime.CheckIndexRange"(i1 [[INDEX_BAD]], i64 [[INDEX]], i1 true, i64 0)
// CHECK: store i64 0, ptr [[KEY:%[-A-Za-z0-9_.]+]]
// CHECK: [[MAP_VALUE_PTR:%.*]] = call ptr @"{{.*}}/runtime/internal/runtime.MapAccess1"(ptr @"map[_llgo_int]_llgo_int", ptr null, ptr [[KEY]])
// CHECK: [[MAP_VALUE:%.*]] = load i64, ptr [[MAP_VALUE_PTR]]
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 [[MAP_VALUE]])

// CHECK-LABEL: define void @"main.main$1"(){{.*}} {
// CHECK: call void @"{{.*}}/runtime/internal/runtime.BindRecoverFrame"(ptr @"main.main$1", ptr [[RECOVER_SLOT:%.*]])
// CHECK: [[RECOVERED:%.*]] = call %"{{.*}}eface" @"{{.*}}/runtime/internal/runtime.Recover"(ptr [[RECOVER_SLOT]])
// CHECK: store %"{{.*}}eface" [[RECOVERED]], ptr %{{.*}}
// CHECK: call { i64, %"{{.*}}iface" } @fmt.Println

func main() {
	defer func() {
		err := recover()
		fmt.Println(err)
	}()
	m := [0]map[int]int{}[a][0]
	print(m)
}
