// LITTEST
package main

import (
	"github.com/goplus/lib/c"
)

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: [[MAP:%[0-9]+]] = call ptr @"{{.*}}MakeMap"(ptr @"map[_llgo_int]_llgo_int", i64 2)
// CHECK-NEXT: [[VALUE100:%[0-9]+]] = call ptr @"{{.*}}MapAssignFast64"(ptr @"map[_llgo_int]_llgo_int", ptr [[MAP]], i64 23)
// CHECK-NEXT: store i64 100, ptr [[VALUE100]]
// CHECK-NEXT: [[VALUE29:%[0-9]+]] = call ptr @"{{.*}}MapAssignFast64"(ptr @"map[_llgo_int]_llgo_int", ptr [[MAP]], i64 7)
// CHECK-NEXT: store i64 29, ptr [[VALUE29]]
// CHECK-NEXT: [[LOOKUP_ADDR:%[0-9]+]] = call ptr @"{{.*}}MapAccess1Fast64"(ptr @"map[_llgo_int]_llgo_int", ptr [[MAP]], i64 23)
// CHECK-NEXT: [[LOOKUP:%[0-9]+]] = load i64, ptr [[LOOKUP_ADDR]]
// CHECK-NEXT: call i32 (ptr, ...) @printf(ptr @{{[0-9]+}}, i64 [[LOOKUP]])
func main() {
	a := map[int]int{23: 100, 7: 29}
	c.Printf(c.Str("Hello %d\n"), a[23])
}
