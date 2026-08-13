// LITTEST
package main

var my = demo2{}.encode

type demo1 struct {
}

func (se demo1) encode() int {
	return 1
}

type demo2 struct {
}

func (se demo2) encode() int {
	return 2
}

func main() {
	se := demo1{}
	f := se.encode
	if f() != 1 {
		panic("error")
	}
}

// Pointer receiver wrappers must guard nil before calling the value receiver.
// CHECK-LABEL: define i64 @"main.(*demo1).encode"(ptr %0){{.*}} {
// CHECK: %[[D1_NIL:[0-9]+]] = icmp eq ptr %0, null
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PanicWrapNilPointer"(i1 %[[D1_NIL]],{{.*}})
// CHECK: call i64 @main.demo1.encode(%main.demo1 zeroinitializer)

// CHECK-LABEL: define i64 @"main.(*demo2).encode"(ptr %0){{.*}} {
// CHECK: %[[D2_NIL:[0-9]+]] = icmp eq ptr %0, null
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PanicWrapNilPointer"(i1 %[[D2_NIL]],{{.*}})
// CHECK: call i64 @main.demo2.encode(%main.demo2 zeroinitializer)

// The package-global method value stores both the selected bound wrapper and
// the zero-sized receiver in init.
// CHECK-LABEL: define void @main.init(){{.*}} {
// CHECK: store { ptr, ptr } { ptr @"main.demo2.encode$bound", ptr @"__llgo.moduleZeroSizedAlloc$" }, ptr @main.my

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: %[[RESULT:[0-9]+]] = call i64 @"main.demo1.encode$bound"(ptr {{(nest|swiftself)}} @"__llgo.moduleZeroSizedAlloc$")
// CHECK: %[[BAD:[0-9]+]] = icmp ne i64 %[[RESULT]], 1
// CHECK: br i1 %[[BAD]]

// CHECK-LABEL: define i64 @"main.demo2.encode$bound"(ptr {{(nest|swiftself)}} %0){{.*}} {
// CHECK: call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %{{[0-9]+}})
// CHECK: %[[D2_RESULT:[0-9]+]] = call i64 @main.demo2.encode(%main.demo2 zeroinitializer)
// CHECK: ret i64 %[[D2_RESULT]]

// CHECK-LABEL: define i64 @"main.demo1.encode$bound"(ptr {{(nest|swiftself)}} %0){{.*}} {
// CHECK: call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %{{[0-9]+}})
// CHECK: %[[D1_RESULT:[0-9]+]] = call i64 @main.demo1.encode(%main.demo1 zeroinitializer)
// CHECK: ret i64 %[[D1_RESULT]]
