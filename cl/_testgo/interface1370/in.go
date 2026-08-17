// LITTEST
package main

import (
	"github.com/xgo-dev/llgo/cl/_testdata/geometry1370"
)

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: %[[RECT:[0-9]+]] = call ptr @"{{.*}}geometry1370.NewRectangle"(double 5.000000e+00, double 3.000000e+00)
// CHECK: %[[ITAB:[0-9]+]] = call ptr @"{{.*}}NewItab"(ptr @"{{.*}}geometry1370.iface{{.*}}", ptr @"*_llgo_{{.*}}geometry1370.Rectangle")
// CHECK: %[[IFACE_TYPE:[0-9]+]] = insertvalue %"{{.*}}iface" undef, ptr %[[ITAB]], 0
// CHECK: %[[RECT_IFACE:[0-9]+]] = insertvalue %"{{.*}}iface" %[[IFACE_TYPE]], ptr %[[RECT]], 1
// CHECK: call void @"{{.*}}geometry1370.RegisterShape"(%"{{.*}}iface" %[[RECT_IFACE]], i64 42)
// CHECK: %[[ID:[0-9]+]] = call i64 @"{{.*}}geometry1370.(*Rectangle).GetID"(ptr %[[RECT]])
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %[[ID]])
func main() {
	rect := geometry1370.NewRectangle(5.0, 3.0)
	geometry1370.RegisterShape(rect, 42)
	println("ID:", rect.GetID())
}
