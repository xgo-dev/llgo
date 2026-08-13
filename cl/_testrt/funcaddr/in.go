// LITTEST
package main

import (
	"unsafe"

	"github.com/goplus/lib/c"
)

//llgo:type C
type Add func(int, int) int

// CHECK-LABEL: define i64 @main.add(i64 %0, i64 %1){{.*}} {
// CHECK: [[ADD_RESULT:%[0-9]+]] = add i64 %0, %1
// CHECK-NEXT: ret i64 [[ADD_RESULT]]
func add(a, b int) int {
	return a + b
}

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: store ptr @main.add, ptr [[DECLARED_SLOT:%[0-9]+]]
// CHECK: store ptr @"main.main$1", ptr [[LITERAL_SLOT:%[0-9]+]]
// CHECK: [[DECLARED_CODE:%[0-9]+]] = load ptr, ptr [[DECLARED_SLOT]]
// CHECK-NEXT: [[DECLARED_MATCH:%[0-9]+]] = icmp eq ptr @main.add, [[DECLARED_CODE]]
// CHECK: [[LITERAL_CODE1:%[0-9]+]] = load ptr, ptr [[LITERAL_SLOT]]
// CHECK-NEXT: [[LITERAL_CODE2:%[0-9]+]] = load ptr, ptr [[LITERAL_SLOT]]
// CHECK-NEXT: [[LITERAL_MATCH:%[0-9]+]] = icmp eq ptr [[LITERAL_CODE1]], [[LITERAL_CODE2]]
func main() {
	var fn Add = add
	var myfn Add = func(a, b int) int {
		return a + b
	}
	println(c.Func(add) == c.Func(fn))
	println(c.Func(fn) == *(*unsafe.Pointer)(unsafe.Pointer(&fn)))
	println(c.Func(myfn) == *(*unsafe.Pointer)(unsafe.Pointer(&myfn)))
}

// CHECK-LABEL: define i64 @"main.main$1"(i64 %0, i64 %1){{.*}} {
// CHECK: [[LITERAL_SUM:%[0-9]+]] = add i64 %0, %1
// CHECK-NEXT: ret i64 [[LITERAL_SUM]]
