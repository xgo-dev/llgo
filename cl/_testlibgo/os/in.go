// LITTEST
package main

import "os"

// CHECK-LABEL: define void @main.main(){{.*}} {
func main() {
	// CHECK: [[GETWD:%[0-9]+]] = call { %"{{.*}}String", %"{{.*}}iface" } @os.Getwd()
	// CHECK-NEXT: [[CWD:%[0-9]+]] = extractvalue { %"{{.*}}String", %"{{.*}}iface" } [[GETWD]], 0
	// CHECK-NEXT: [[GETWD_ERR:%[0-9]+]] = extractvalue { %"{{.*}}String", %"{{.*}}iface" } [[GETWD]], 1
	// CHECK: [[ERR_TYPE:%[0-9]+]] = call ptr @"{{.*}}IfaceType"(%"{{.*}}iface" [[GETWD_ERR]])
	// CHECK: [[ERR_EFACE:%[0-9]+]] = insertvalue %"{{.*}}eface" %{{[0-9]+}}, ptr %{{[0-9]+}}, 1
	// CHECK: [[NO_ERR_TYPE:%[0-9]+]] = call ptr @"{{.*}}IfaceType"(%"{{.*}}iface" zeroinitializer)
	// CHECK: [[NO_ERR:%[0-9]+]] = insertvalue %"{{.*}}eface" %{{[0-9]+}}, ptr null, 1
	// CHECK-NEXT: [[ERR_IS_NIL:%[0-9]+]] = call i1 @"{{.*}}EfaceEqual"(%"{{.*}}eface" [[ERR_EFACE]], %"{{.*}}eface" [[NO_ERR]])
	// CHECK-NEXT: [[HAS_ERR:%[0-9]+]] = xor i1 [[ERR_IS_NIL]], true
	// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}String" [[CWD]])
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	println("cwd:", wd)
}
