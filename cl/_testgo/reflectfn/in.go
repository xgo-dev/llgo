// LITTEST
package main

import (
	"fmt"
	"reflect"
)

// Capturing and declared functions both enter reflect as closure pairs, but a
// declared function has a nil environment.
// CHECK-LABEL: define void @main.demo(){{.*}} {
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}runtime.String" { ptr @{{[0-9]+}}, i64 4 })
// CHECK: ret void
// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: %[[CAPTURED_VALUE:[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT: store i64 100, ptr %[[CAPTURED_VALUE]]
// CHECK: %[[ENV:[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT: %[[ENV_SLOT:[0-9]+]] = getelementptr inbounds { ptr }, ptr %[[ENV]], i32 0, i32 0
// CHECK-NEXT: store ptr %[[CAPTURED_VALUE]], ptr %[[ENV_SLOT]]
// CHECK-NEXT: %[[CAPTURED_FN:[0-9]+]] = insertvalue { ptr, ptr } { ptr @"main.main$1", ptr undef }, ptr %[[ENV]], 1
// CHECK: store { ptr, ptr } %[[CAPTURED_FN]], ptr %{{[0-9]+}}
// CHECK: store { ptr, ptr } { ptr @main.demo, ptr null }, ptr %{{[0-9]+}}
// CHECK: store { ptr, ptr } { ptr @main.demo, ptr null }, ptr %{{[0-9]+}}
// CHECK: store { ptr, ptr } %[[CAPTURED_FN]], ptr %{{[0-9]+}}
// CHECK: %[[CAPTURED_EFACE:[0-9]+]] = insertvalue %"{{.*}}runtime.eface" { ptr @{{.*}}, ptr undef }, ptr %{{[0-9]+}}, 1
// CHECK-NEXT: %[[CAPTURED_REFLECT:[0-9]+]] = call %reflect.Value @reflect.ValueOf(%"{{.*}}runtime.eface" %[[CAPTURED_EFACE]])
// CHECK-NEXT: %[[CAPTURED_PTR:[0-9]+]] = call ptr @reflect.Value.UnsafePointer(%reflect.Value %[[CAPTURED_REFLECT]])
// CHECK: store { ptr, ptr } { ptr @main.demo, ptr null }, ptr %[[DECL_BOX1:[0-9]+]]
// CHECK-NEXT: %[[DECL_EFACE1:[0-9]+]] = insertvalue %"{{.*}}runtime.eface" { ptr @{{.*}}, ptr undef }, ptr %[[DECL_BOX1]], 1
// CHECK-NEXT: %[[DECL_REFLECT1:[0-9]+]] = call %reflect.Value @reflect.ValueOf(%"{{.*}}runtime.eface" %[[DECL_EFACE1]])
// CHECK-NEXT: %[[DECL_PTR1:[0-9]+]] = call ptr @reflect.Value.UnsafePointer(%reflect.Value %[[DECL_REFLECT1]])
// CHECK: store { ptr, ptr } { ptr @main.demo, ptr null }, ptr %[[DECL_BOX2:[0-9]+]]
// CHECK-NEXT: %[[DECL_EFACE2:[0-9]+]] = insertvalue %"{{.*}}runtime.eface" { ptr @{{.*}}, ptr undef }, ptr %[[DECL_BOX2]], 1
// CHECK-NEXT: %[[DECL_REFLECT2:[0-9]+]] = call %reflect.Value @reflect.ValueOf(%"{{.*}}runtime.eface" %[[DECL_EFACE2]])
// CHECK-NEXT: %[[DECL_PTR2:[0-9]+]] = call ptr @reflect.Value.UnsafePointer(%reflect.Value %[[DECL_REFLECT2]])
// CHECK-LABEL: define void @"main.main$1"(ptr {{(nest|swiftself)}} %0){{.*}} {
// CHECK: %[[ENV_VALUE:[0-9]+]] = load { ptr }, ptr %0
// CHECK-NEXT: %[[VALUE_PTR:[0-9]+]] = extractvalue { ptr } %[[ENV_VALUE]], 0
// CHECK-NEXT: %[[VALUE:[0-9]+]] = load i64, ptr %[[VALUE_PTR]]
// CHECK-NEXT: call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %[[VALUE]])

func demo() {
	println("demo")
}

func main() {
	v := 100
	fn := func() {
		println(v)
	}
	fdemo := demo
	fmt.Println(fn)
	fmt.Println(demo)
	fmt.Println(fdemo)
	fmt.Println(reflect.ValueOf(fn).UnsafePointer())
	fmt.Println(reflect.ValueOf(demo).UnsafePointer())
	fmt.Println(reflect.ValueOf(fdemo).UnsafePointer())
}
