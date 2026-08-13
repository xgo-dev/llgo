// LITTEST
package main

import "net/textproto"

// CHECK-LABEL: define void @main.main(){{.*}} {
func main() {
	// CHECK: [[HEADER:%[0-9]+]] = call %"{{.*}}/runtime/internal/runtime.String" @"net/textproto.CanonicalMIMEHeaderKey"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 4 })
	// CHECK-NEXT: call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" [[HEADER]])
	println(textproto.CanonicalMIMEHeaderKey("host"))
}
