// LITTEST
package main

import (
	"github.com/goplus/llgo/cl/_testdata/geometry1370"
)

// CHECK-LABEL: define void @"{{.*}}interface1370.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}geometry1370.NewRectangle"(double 5.000000e+00, double 3.000000e+00)
// CHECK-NEXT:   %1 = call ptr @"{{.*}}NewItab"(ptr @"{{.*}}geometry1370.iface{{.*}}", ptr @"*_llgo_{{.*}}geometry1370.Rectangle")
// CHECK-NEXT:   %2 = insertvalue %"{{.*}}iface" undef, ptr %1, 0
// CHECK-NEXT:   %3 = insertvalue %"{{.*}}iface" %2, ptr %0, 1
// CHECK-NEXT:   call void @"{{.*}}geometry1370.RegisterShape"(%"{{.*}}iface" %3, i64 42)
// CHECK-NEXT:   %4 = call i64 @"{{.*}}geometry1370.(*Rectangle).GetID"(ptr %0)
// CHECK-NEXT:   call void @"{{.*}}PrintString"(%"{{.*}}String" { ptr @{{[0-9]+}}, i64 3 })
// CHECK-NEXT:   call void @"{{.*}}PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}PrintInt"(i64 %4)
// CHECK-NEXT:   call void @"{{.*}}PrintByte"(i8 10)
// CHECK-NEXT:   ret void
func main() {
	rect := geometry1370.NewRectangle(5.0, 3.0)
	geometry1370.RegisterShape(rect, 42)
	println("ID:", rect.GetID())
}
