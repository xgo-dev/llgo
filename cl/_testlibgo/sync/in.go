// LITTEST
package main

import (
	"sync"
)

// The closure passed to Once.Do must own the argument string, and the callback
// must recover that exact capture before printing it.
// CHECK-LABEL: define void @main.f(%"{{.*}}String" %0){{.*}} {
// CHECK: [[STRING_ADDR:%[0-9]+]] = call ptr @"{{.*}}AllocZ"(i64 16)
// CHECK-NEXT: store %"{{.*}}String" %0, ptr [[STRING_ADDR]]
// CHECK-NEXT: [[ENV:%[0-9]+]] = call ptr @"{{.*}}AllocU"(i64 8)
// CHECK-NEXT: [[CAPTURE:%[0-9]+]] = getelementptr inbounds { ptr }, ptr [[ENV]], i32 0, i32 0
// CHECK-NEXT: store ptr [[STRING_ADDR]], ptr [[CAPTURE]]
// CHECK-NEXT: [[CLOSURE:%[0-9]+]] = insertvalue { ptr, ptr } { ptr @"main.f$1", ptr undef }, ptr [[ENV]], 1
// CHECK-NEXT: call void @"sync.(*Once).Do"(ptr @main.once, { ptr, ptr } [[CLOSURE]])
// CHECK-LABEL: define void @"main.f$1"(ptr {{(nest|swiftself)}} %0){{.*}} {
// CHECK: [[CAPTURED_ENV:%[0-9]+]] = load { ptr }, ptr %0
// CHECK-NEXT: [[CAPTURED_ADDR:%[0-9]+]] = extractvalue { ptr } [[CAPTURED_ENV]], 0
// CHECK-NEXT: [[CAPTURED_STRING:%[0-9]+]] = load %"{{.*}}String", ptr [[CAPTURED_ADDR]]
// CHECK-NEXT: call void @"{{.*}}PrintString"(%"{{.*}}String" [[CAPTURED_STRING]])
// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: call void @main.f(%"{{.*}}String" { ptr @{{[0-9]+}}, i64 7 })
// CHECK-NEXT: call void @main.f(%"{{.*}}String" { ptr @{{[0-9]+}}, i64 8 })

var once sync.Once

func f(s string) {
	once.Do(func() {
		println(s)
	})
}

func main() {
	f("Do once")
	f("Do twice")
}
