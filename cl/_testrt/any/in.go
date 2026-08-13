// LITTEST
package main

import (
	"github.com/goplus/lib/c"
)

func hi(a any) *c.Char {
	return a.(*c.Char)
}

func incVal(a any) int {
	return a.(int) + 1
}

func main() {
	c.Printf(c.Str("%s %d\n"), hi(c.Str("Hello")), incVal(100))
}

// CHECK-LABEL: define ptr @main.hi(%"{{.*}}eface" %0){{.*}} {
// CHECK: [[HI_TYPE:%.*]] = extractvalue %"{{.*}}eface" %0, 0
// CHECK-NEXT: [[HI_MATCH:%.*]] = icmp eq ptr [[HI_TYPE]], @"*_llgo_int8"
// CHECK-NEXT: br i1 [[HI_MATCH]], label %{{.*}}, label %{{.*}}
// CHECK: [[HI_DATA:%.*]] = extractvalue %"{{.*}}eface" %0, 1
// CHECK-NEXT: ret ptr [[HI_DATA]]
// CHECK: call void @"{{.*}}PanicTypeAssert"(ptr null, ptr [[HI_TYPE]], ptr @"*_llgo_int8")
// CHECK-NEXT: unreachable

// CHECK-LABEL: define i64 @main.incVal(%"{{.*}}eface" %0){{.*}} {
// CHECK: [[INC_TYPE:%.*]] = extractvalue %"{{.*}}eface" %0, 0
// CHECK-NEXT: [[INC_MATCH:%.*]] = icmp eq ptr [[INC_TYPE]], @_llgo_int
// CHECK-NEXT: br i1 [[INC_MATCH]], label %{{.*}}, label %{{.*}}
// CHECK: [[INC_DATA:%.*]] = extractvalue %"{{.*}}eface" %0, 1
// CHECK-NEXT: [[INC_VALUE:%.*]] = load i64, ptr [[INC_DATA]]
// CHECK-NEXT: [[INC_RESULT:%.*]] = add i64 [[INC_VALUE]], 1
// CHECK-NEXT: ret i64 [[INC_RESULT]]
// CHECK: call void @"{{.*}}PanicTypeAssert"(ptr null, ptr [[INC_TYPE]], ptr @_llgo_int)
// CHECK-NEXT: unreachable

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: [[HI_RESULT:%.*]] = call ptr @main.hi(%"{{.*}}eface" { ptr @"*_llgo_int8", ptr @{{[0-9]+}} })
// CHECK: [[INT_DATA:%.*]] = call ptr @"{{.*}}AllocU"(i64 8)
// CHECK-NEXT: store i64 100, ptr [[INT_DATA]]
// CHECK-NEXT: [[INT_EFACE:%.*]] = insertvalue %"{{.*}}eface" { ptr @_llgo_int, ptr undef }, ptr [[INT_DATA]], 1
// CHECK-NEXT: [[INC_CALL_RESULT:%.*]] = call i64 @main.incVal(%"{{.*}}eface" [[INT_EFACE]])
// CHECK-NEXT: call i32 (ptr, ...) @printf(ptr @{{[0-9]+}}, ptr [[HI_RESULT]], i64 [[INC_CALL_RESULT]])
