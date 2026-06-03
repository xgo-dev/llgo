// LITTEST
package main

import "errors"

// CHECK-LABEL: define void @"{{.*}}/cl/_testlibgo/errors.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call %"{{.*}}/runtime/internal/runtime.iface" @errors.New(%"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 5 })
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %0)
// CHECK-NEXT:   %2 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %0, 1
// CHECK-NEXT:   %3 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" undef, ptr %1, 0
// CHECK-NEXT:   %4 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" %3, ptr %2, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %4)
// CHECK-NEXT:   unreachable
// CHECK-NEXT: }
func main() {
	err := errors.New("error")
	panic(err)
}
