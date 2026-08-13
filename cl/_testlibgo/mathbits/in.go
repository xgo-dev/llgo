// LITTEST
package main

import (
	"math/bits"
)

// CHECK-LABEL: define void @main.main(){{.*}} {
func main() {
	// CHECK: [[LEN8:%[0-9]+]] = call i64 @"math/bits.Len8"(i8 20)
	// CHECK-NEXT: call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 [[LEN8]])
	// CHECK: [[ONES:%[0-9]+]] = call i64 @"math/bits.OnesCount"(i64 20)
	// CHECK-NEXT: call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 [[ONES]])
	println(bits.Len8(20))
	println(bits.OnesCount(20))
}
