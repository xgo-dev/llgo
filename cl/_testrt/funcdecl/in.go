// LITTEST
package main

import (
	"unsafe"
)

// A declared function and its parameter retain the same canonical descriptor.
// Interface assertions must retain the Go closure role independently of storage.
// CHECK: [[DECL_DESCRIPTOR:@"_llgo_funcval\$[0-9a-f]+"]] = linkonce_odr constant { ptr, ptr } { ptr @main.demo, ptr null }
// CHECK-LABEL: define void @main.check(ptr %0){{.*}} {
// CHECK: %[[ARG_EFACE:[0-9]+]] = insertvalue %"{{.*}}runtime.eface" { ptr @[[CLOSURE_TYPE:"_llgo_closure\$[^"]+"]], ptr undef }, ptr %0, 1
// CHECK: %[[DECL_MATCH:[0-9]+]] = call i1 @"{{.*}}/runtime/internal/runtime.MatchesClosure"(ptr @[[CLOSURE_TYPE]], ptr @[[CLOSURE_TYPE]])
// CHECK: %[[ARG_TYPE:[0-9]+]] = extractvalue %"{{.*}}runtime.eface" %[[ARG_EFACE]], 0
// CHECK-NEXT: %[[ARG_MATCH:[0-9]+]] = call i1 @"{{.*}}/runtime/internal/runtime.MatchesClosure"(ptr @[[CLOSURE_TYPE]], ptr %[[ARG_TYPE]])
// CHECK: %[[DECL_PTR:[0-9]+]] = call ptr @main.closurePtr(%"{{.*}}runtime.eface" { ptr @[[CLOSURE_TYPE]], ptr [[DECL_DESCRIPTOR]] })
// CHECK-NEXT: %[[ARG_PTR:[0-9]+]] = call ptr @main.closurePtr(%"{{.*}}runtime.eface" %[[ARG_EFACE]])
// CHECK-NEXT: %[[SAME_PTR:[0-9]+]] = icmp eq ptr %[[DECL_PTR]], %[[ARG_PTR]]
// CHECK-NEXT: call void @"{{.*}}/runtime/internal/runtime.PrintBool"(i1 %[[SAME_PTR]])
// CHECK-LABEL: define ptr @main.closurePtr(%"{{.*}}runtime.eface" %0){{.*}} {
// CHECK: %[[EFACE_BOX:[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT: store %"{{.*}}runtime.eface" %0, ptr %[[EFACE_BOX]]
// CHECK: %[[CLOSURE_SLOT:[0-9]+]] = getelementptr inbounds nuw %main.rtype, ptr %[[EFACE_BOX]], i32 0, i32 1
// CHECK-NEXT: %[[CLOSURE:[0-9]+]] = load ptr, ptr %[[CLOSURE_SLOT]]
// CHECK-NEXT: %[[CODE_SLOT:[0-9]+]] = getelementptr inbounds nuw { ptr, ptr }, ptr %[[CLOSURE]], i32 0, i32 0
// CHECK-NEXT: %[[CODE:[0-9]+]] = load ptr, ptr %[[CODE_SLOT]]
// CHECK-NEXT: ret ptr %[[CODE]]
// CHECK-LABEL: define void @main.demo(){{.*}} {
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}runtime.String" { ptr @{{[0-9]+}}, i64 4 })
// CHECK: ret void
// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: call void @main.check(ptr [[DECL_DESCRIPTOR]])

func check(fn func()) {
	var a any = demo
	var b any = fn
	fn1 := a.(func())
	fn2 := b.(func())
	println(a, b, fn, fn1, fn2, demo)
	println(closurePtr(a) == closurePtr(b))
}

func closurePtr(a any) unsafe.Pointer {
	return (*rtype)(unsafe.Pointer(&a)).ptr.fn
}

type rtype struct {
	typ unsafe.Pointer
	ptr *struct {
		fn  unsafe.Pointer
		env unsafe.Pointer
	}
}

func demo() {
	println("demo")
}

func main() {
	println("hello")
	check(demo)
}
