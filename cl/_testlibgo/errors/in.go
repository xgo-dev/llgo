// LITTEST
package main

import "errors"

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: [[ERR:%[0-9]+]] = call %"{{.*}}iface" @errors.New(%"{{.*}}String" { ptr @{{[0-9]+}}, i64 5 })
// CHECK-NEXT: [[ERR_TYPE:%[0-9]+]] = call ptr @"{{.*}}IfaceType"(%"{{.*}}iface" [[ERR]])
// CHECK-NEXT: [[ERR_DATA:%[0-9]+]] = extractvalue %"{{.*}}iface" [[ERR]], 1
// CHECK-NEXT: [[PANIC0:%[0-9]+]] = insertvalue %"{{.*}}eface" undef, ptr [[ERR_TYPE]], 0
// CHECK-NEXT: [[PANIC_VALUE:%[0-9]+]] = insertvalue %"{{.*}}eface" [[PANIC0]], ptr [[ERR_DATA]], 1
// CHECK-NEXT: call void @"{{.*}}Panic"(%"{{.*}}eface" [[PANIC_VALUE]])
func main() {
	err := errors.New("error")
	panic(err)
}
