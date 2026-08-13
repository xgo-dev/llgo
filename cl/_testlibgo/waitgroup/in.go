// LITTEST
package main

import (
	"sync"
)

// CHECK-LABEL: define void @main.main(){{.*}} {
// Both goroutines capture the same WaitGroup that Add and Wait operate on.
// CHECK: [[WG:%.*]] = call ptr @"{{.*}}AllocZ"(i64 16)
// CHECK-NEXT: call void @"sync.(*WaitGroup).Add"(ptr [[WG]], i64 2)
// CHECK: [[WG1_ENV:%.*]] = call ptr @"{{.*}}AllocU"(i64 8)
// CHECK: store ptr [[WG]], ptr {{%.*}}
// CHECK: [[WG1_CLOSURE:%.*]] = insertvalue { ptr, ptr } { ptr @"main.main$1", ptr undef }, ptr [[WG1_ENV]], 1
// CHECK: store { ptr, ptr } [[WG1_CLOSURE]], ptr {{%.*}}
// CHECK: call void @"{{.*}}NewProc"(ptr @"main._llgo_routine$1", ptr {{%.*}}, i64 0)
// CHECK: [[WG2_ENV:%.*]] = call ptr @"{{.*}}AllocU"(i64 8)
// CHECK: store ptr [[WG]], ptr {{%.*}}
// CHECK: [[WG2_CLOSURE:%.*]] = insertvalue { ptr, ptr } { ptr @"main.main$2", ptr undef }, ptr [[WG2_ENV]], 1
// CHECK: store { ptr, ptr } [[WG2_CLOSURE]], ptr {{%.*}}
// CHECK: call void @"{{.*}}NewProc"(ptr @"main._llgo_routine$2", ptr {{%.*}}, i64 0)
// CHECK-NEXT: call void @"sync.(*WaitGroup).Wait"(ptr [[WG]])

// Each worker stores its captured WaitGroup in a defer node and calls Done only
// after its work body, using the value recovered from that same node.
// CHECK-LABEL: define void @"main.main$1"(ptr {{(nest|swiftself)}} %0){{.*}} {
// CHECK: [[WG1_CAPTURE:%.*]] = load { ptr }, ptr %0
// CHECK-NEXT: [[WG1_VALUE:%.*]] = extractvalue { ptr } [[WG1_CAPTURE]], 0
// CHECK: [[WG1_HEAD:%.*]] = getelementptr inbounds %"{{.*}}Defer", ptr {{%.*}}, i32 0, i32 5
// CHECK: [[WG1_NODE:%.*]] = call ptr @"{{.*}}AllocU"(i64 24)
// CHECK: [[WG1_NODE_DATA:%.*]] = getelementptr inbounds { ptr, i64, ptr }, ptr [[WG1_NODE]], i32 0, i32 2
// CHECK-NEXT: store ptr [[WG1_VALUE]], ptr [[WG1_NODE_DATA]]
// CHECK-NEXT: store ptr [[WG1_NODE]], ptr [[WG1_HEAD]]
// CHECK: call void @"{{.*}}PrintString"(%"{{.*}}String" { ptr @{{[0-9]+}}, i64 6 })
// CHECK: [[WG1_ACTIVE:%.*]] = load ptr, ptr [[WG1_HEAD]]
// CHECK-NEXT: [[WG1_NODE_VALUE:%.*]] = load { ptr, i64, ptr }, ptr [[WG1_ACTIVE]]
// CHECK: [[WG1_DONE_VALUE:%.*]] = extractvalue { ptr, i64, ptr } [[WG1_NODE_VALUE]], 2
// CHECK-NEXT: call void @"{{.*}}FreeDeferNode"(ptr [[WG1_ACTIVE]])
// CHECK-NEXT: call void @"sync.(*WaitGroup).Done"(ptr [[WG1_DONE_VALUE]])

// CHECK-LABEL: define void @"main.main$2"(ptr {{(nest|swiftself)}} %0){{.*}} {
// CHECK: [[WG2_CAPTURE:%.*]] = load { ptr }, ptr %0
// CHECK-NEXT: [[WG2_VALUE:%.*]] = extractvalue { ptr } [[WG2_CAPTURE]], 0
// CHECK: [[WG2_HEAD:%.*]] = getelementptr inbounds %"{{.*}}Defer", ptr {{%.*}}, i32 0, i32 5
// CHECK: [[WG2_NODE:%.*]] = call ptr @"{{.*}}AllocU"(i64 24)
// CHECK: [[WG2_NODE_DATA:%.*]] = getelementptr inbounds { ptr, i64, ptr }, ptr [[WG2_NODE]], i32 0, i32 2
// CHECK-NEXT: store ptr [[WG2_VALUE]], ptr [[WG2_NODE_DATA]]
// CHECK-NEXT: store ptr [[WG2_NODE]], ptr [[WG2_HEAD]]
// CHECK: call void @"{{.*}}PrintString"(%"{{.*}}String" { ptr @{{[0-9]+}}, i64 6 })
// CHECK: [[WG2_ACTIVE:%.*]] = load ptr, ptr [[WG2_HEAD]]
// CHECK-NEXT: [[WG2_NODE_VALUE:%.*]] = load { ptr, i64, ptr }, ptr [[WG2_ACTIVE]]
// CHECK: [[WG2_DONE_VALUE:%.*]] = extractvalue { ptr, i64, ptr } [[WG2_NODE_VALUE]], 2
// CHECK-NEXT: call void @"{{.*}}FreeDeferNode"(ptr [[WG2_ACTIVE]])
// CHECK-NEXT: call void @"sync.(*WaitGroup).Done"(ptr [[WG2_DONE_VALUE]])

func main() {
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		println("work 1")
	}()
	go func() {
		defer wg.Done()
		println("work 2")
	}()
	wg.Wait()
	println("done")
}
