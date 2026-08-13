// LITTEST
package main

import (
	"github.com/goplus/lib/c"
)

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: [[MAP:%[0-9]+]] = call ptr @"{{.*}}MakeMap"(ptr @"map[_llgo_int]_llgo_int", i64 2)
// CHECK-NEXT: [[KEY23:%[0-9]+]] = call ptr @"{{.*}}AllocU"(i64 8)
// CHECK-NEXT: store i64 23, ptr [[KEY23]]
// CHECK-NEXT: [[VALUE100:%[0-9]+]] = call ptr @"{{.*}}MapAssign"(ptr @"map[_llgo_int]_llgo_int", ptr [[MAP]], ptr [[KEY23]])
// CHECK-NEXT: store i64 100, ptr [[VALUE100]]
// CHECK-NEXT: [[KEY7:%[0-9]+]] = call ptr @"{{.*}}AllocU"(i64 8)
// CHECK-NEXT: store i64 7, ptr [[KEY7]]
// CHECK-NEXT: [[VALUE29:%[0-9]+]] = call ptr @"{{.*}}MapAssign"(ptr @"map[_llgo_int]_llgo_int", ptr [[MAP]], ptr [[KEY7]])
// CHECK-NEXT: store i64 29, ptr [[VALUE29]]
// CHECK-NEXT: [[LOOKUP_KEY:%[0-9]+]] = call ptr @"{{.*}}AllocU"(i64 8)
// CHECK-NEXT: store i64 23, ptr [[LOOKUP_KEY]]
// CHECK-NEXT: [[LOOKUP_ADDR:%[0-9]+]] = call ptr @"{{.*}}MapAccess1"(ptr @"map[_llgo_int]_llgo_int", ptr [[MAP]], ptr [[LOOKUP_KEY]])
// CHECK-NEXT: [[LOOKUP:%[0-9]+]] = load i64, ptr [[LOOKUP_ADDR]]
// CHECK-NEXT: call i32 (ptr, ...) @printf(ptr @{{[0-9]+}}, i64 [[LOOKUP]])
func main() {
	a := map[int]int{23: 100, 7: 29}
	c.Printf(c.Str("Hello %d\n"), a[23])
}
