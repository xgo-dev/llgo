// LITTEST
package main

// CHECK: {{^}}@0 = private unnamed_addr constant [52 x i8] c"{{.*}}/cl/_testrt/closurebound.demo1", align 1{{$}}
// CHECK: {{^}}@1 = private unnamed_addr constant [6 x i8] c"encode", align 1{{$}}
// CHECK: {{^}}@2 = private unnamed_addr constant [52 x i8] c"{{.*}}/cl/_testrt/closurebound.demo2", align 1{{$}}
// CHECK: {{^}}@3 = private unnamed_addr constant [5 x i8] c"error", align 1{{$}}

var my = demo2{}.encode

type demo1 struct {
}

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testrt/closurebound.demo1.encode"(%"{{.*}}/cl/_testrt/closurebound.demo1" %0){{.*}} {
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

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testrt/closurebound.(*demo1).encode"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PanicWrapNilPointer"(i1 %1, %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 52 }, %"{{.*}}/runtime/internal/runtime.String" { ptr @1, i64 6 })
// CHECK-NEXT:   %2 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %2)
// CHECK-NEXT:   %3 = call i64 @"{{.*}}/cl/_testrt/closurebound.demo1.encode"(%"{{.*}}/cl/_testrt/closurebound.demo1" zeroinitializer)
// CHECK-NEXT:   ret i64 %3
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testrt/closurebound.demo2.encode"(%"{{.*}}/cl/_testrt/closurebound.demo2" %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   ret i64 2
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testrt/closurebound.(*demo2).encode"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PanicWrapNilPointer"(i1 %1, %"{{.*}}/runtime/internal/runtime.String" { ptr @2, i64 52 }, %"{{.*}}/runtime/internal/runtime.String" { ptr @1, i64 6 })
// CHECK-NEXT:   %2 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %2)
// CHECK-NEXT:   %3 = call i64 @"{{.*}}/cl/_testrt/closurebound.demo2.encode"(%"{{.*}}/cl/_testrt/closurebound.demo2" zeroinitializer)
// CHECK-NEXT:   ret i64 %3
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/closurebound.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testrt/closurebound.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testrt/closurebound.init$guard", align 1
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 0)
// CHECK-NEXT:   %2 = getelementptr inbounds { %"{{.*}}/cl/_testrt/closurebound.demo2" }, ptr %1, i32 0, i32 0
// CHECK-NEXT:   store %"{{.*}}/cl/_testrt/closurebound.demo2" zeroinitializer, ptr %2, align 1
// CHECK-NEXT:   %3 = insertvalue { ptr, ptr } { ptr @"{{.*}}/cl/_testrt/closurebound.demo2.encode$bound", ptr undef }, ptr %1, 1
// CHECK-NEXT:   store { ptr, ptr } %3, ptr @"{{.*}}/cl/_testrt/closurebound.my", align 8
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/closurebound.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 0)
// CHECK-NEXT:   %1 = getelementptr inbounds { %"{{.*}}/cl/_testrt/closurebound.demo1" }, ptr %0, i32 0, i32 0
// CHECK-NEXT:   store %"{{.*}}/cl/_testrt/closurebound.demo1" zeroinitializer, ptr %1, align 1
// CHECK-NEXT:   %2 = insertvalue { ptr, ptr } { ptr @"{{.*}}/cl/_testrt/closurebound.demo1.encode$bound", ptr undef }, ptr %0, 1
// CHECK-NEXT:   %3 = extractvalue { ptr, ptr } %2, 1
// CHECK-NEXT:   %4 = extractvalue { ptr, ptr } %2, 0
// CHECK-NEXT:   %5 = call i64 %4(ptr %3)
// CHECK-NEXT:   %6 = icmp ne i64 %5, 1
// CHECK-NEXT:   br i1 %6, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %7 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @3, i64 5 }, ptr %7, align 8
// CHECK-NEXT:   %8 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %7, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %8)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testrt/closurebound.demo2.encode$bound"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %1)
// CHECK-NEXT:   %2 = call i64 @"{{.*}}/cl/_testrt/closurebound.demo2.encode"(%"{{.*}}/cl/_testrt/closurebound.demo2" zeroinitializer)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testrt/closurebound.demo1.encode$bound"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %1)
// CHECK-NEXT:   %2 = call i64 @"{{.*}}/cl/_testrt/closurebound.demo1.encode"(%"{{.*}}/cl/_testrt/closurebound.demo1" zeroinitializer)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }
