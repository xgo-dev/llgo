// LITTEST
package main

// CHECK: {{^}}@0 = private unnamed_addr constant [52 x i8] c"{{.*}}/cl/_testrt/closurebound.demo1", align 1{{$}}
// CHECK: {{^}}@1 = private unnamed_addr constant [6 x i8] c"encode", align 1{{$}}
// CHECK: {{^}}@2 = private unnamed_addr constant [52 x i8] c"{{.*}}/cl/_testrt/closurebound.demo2", align 1{{$}}
// CHECK: {{^}}@3 = private unnamed_addr constant [5 x i8] c"error", align 1{{$}}

var my = demo2{}.encode

type demo1 struct {
}

// CHECK-LABEL: define i64 @main.demo1.encode(%main.demo1 %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   ret i64 1
// CHECK-NEXT: }

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

// CHECK-LABEL: define i64 @"main.(*demo1).encode"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PanicWrapNilPointer"(i1 %1, %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 52 }, %"{{.*}}/runtime/internal/runtime.String" { ptr @1, i64 6 })
// CHECK-NEXT:   %2 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %2)
// CHECK-NEXT:   %3 = call i64 @main.demo1.encode(%main.demo1 zeroinitializer)
// CHECK-NEXT:   ret i64 %3
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @main.demo2.encode(%main.demo2 %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   ret i64 2
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"main.(*demo2).encode"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PanicWrapNilPointer"(i1 %1, %"{{.*}}/runtime/internal/runtime.String" { ptr @2, i64 52 }, %"{{.*}}/runtime/internal/runtime.String" { ptr @1, i64 6 })
// CHECK-NEXT:   %2 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %2)
// CHECK-NEXT:   %3 = call i64 @main.demo2.encode(%main.demo2 zeroinitializer)
// CHECK-NEXT:   ret i64 %3
// CHECK-NEXT: }

// CHECK-LABEL: define void @main.init(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"main.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"main.init$guard", align 1
// CHECK-NEXT:   store { ptr, ptr } { ptr @"main.demo2.encode$bound", ptr @"__llgo.moduleZeroSizedAlloc$" }, ptr @main.my, align 8
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call i64 @"main.demo1.encode$bound"(ptr {{(nest|swiftself)}} @"__llgo.moduleZeroSizedAlloc$")
// CHECK-NEXT:   %1 = icmp ne i64 %0, 1
// CHECK-NEXT:   br i1 %1, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %2 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @3, i64 5 }, ptr %2, align 8
// CHECK-NEXT:   %3 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %2, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %3)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"main.demo2.encode$bound"(ptr {{(nest|swiftself)}} %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %1)
// CHECK-NEXT:   %2 = call i64 @main.demo2.encode(%main.demo2 zeroinitializer)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"main.demo1.encode$bound"(ptr {{(nest|swiftself)}} %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %1)
// CHECK-NEXT:   %2 = call i64 @main.demo1.encode(%main.demo1 zeroinitializer)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }
