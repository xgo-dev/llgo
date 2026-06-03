// LITTEST
package main

import (
	"github.com/goplus/lib/c"
)

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/map.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.MakeMap"(ptr @"map[_llgo_int]_llgo_int", i64 2)
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 23, ptr %1, align 8
// CHECK-NEXT:   %2 = call ptr @"{{.*}}/runtime/internal/runtime.MapAssign"(ptr @"map[_llgo_int]_llgo_int", ptr %0, ptr %1)
// CHECK-NEXT:   store i64 100, ptr %2, align 8
// CHECK-NEXT:   %3 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 7, ptr %3, align 8
// CHECK-NEXT:   %4 = call ptr @"{{.*}}/runtime/internal/runtime.MapAssign"(ptr @"map[_llgo_int]_llgo_int", ptr %0, ptr %3)
// CHECK-NEXT:   store i64 29, ptr %4, align 8
// CHECK-NEXT:   %5 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 23, ptr %5, align 8
// CHECK-NEXT:   %6 = call ptr @"{{.*}}/runtime/internal/runtime.MapAccess1"(ptr @"map[_llgo_int]_llgo_int", ptr %0, ptr %5)
// CHECK-NEXT:   %7 = load i64, ptr %6, align 8
// CHECK-NEXT:   %8 = call i32 (ptr, ...) @printf(ptr @13, i64 %7)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func main() {
	a := map[int]int{23: 100, 7: 29}
	c.Printf(c.Str("Hello %d\n"), a[23])
}
