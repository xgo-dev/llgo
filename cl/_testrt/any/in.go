// LITTEST
package main

import (
	"github.com/goplus/lib/c"
)

// CHECK: @1 = private unnamed_addr constant [29 x i8] c"*github.com/goplus/lib/c.Char", align 1
// CHECK: @2 = private unnamed_addr constant [3 x i8] c"int", align 1
// CHECK: @3 = private unnamed_addr constant [7 x i8] c"%s %d\0A\00", align 1
// CHECK: @4 = private unnamed_addr constant [6 x i8] c"Hello\00", align 1

// CHECK-LABEL: define ptr @"{{.*}}/cl/_testrt/any.hi"(%"{{.*}}/runtime/internal/runtime.eface" %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %0, 0
// CHECK-NEXT:   %2 = icmp eq ptr %1, @"*_llgo_int8"
// CHECK-NEXT:   br i1 %2, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %3 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %0, 1
// CHECK-NEXT:   ret ptr %3
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PanicTypeAssert"(ptr %1, %"{{.*}}/runtime/internal/runtime.String" { ptr @1, i64 29 }, %"{{.*}}/runtime/internal/runtime.String" zeroinitializer)
// CHECK-NEXT:   unreachable
// CHECK-NEXT: }

func hi(a any) *c.Char {
	return a.(*c.Char)
}

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testrt/any.incVal"(%"{{.*}}/runtime/internal/runtime.eface" %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %0, 0
// CHECK-NEXT:   %2 = icmp eq ptr %1, @_llgo_int
// CHECK-NEXT:   br i1 %2, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %3 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %0, 1
// CHECK-NEXT:   %4 = load i64, ptr %3, align 8
// CHECK-NEXT:   %5 = add i64 %4, 1
// CHECK-NEXT:   ret i64 %5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PanicTypeAssert"(ptr %1, %"{{.*}}/runtime/internal/runtime.String" { ptr @2, i64 3 }, %"{{.*}}/runtime/internal/runtime.String" zeroinitializer)
// CHECK-NEXT:   unreachable
// CHECK-NEXT: }

func incVal(a any) int {
	return a.(int) + 1
}

func main() {
	c.Printf(c.Str("%s %d\n"), hi(c.Str("Hello")), incVal(100))
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/any.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testrt/any.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testrt/any.init$guard", align 1
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/any.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/cl/_testrt/any.hi"(%"{{.*}}/runtime/internal/runtime.eface" { ptr @"*_llgo_int8", ptr @4 })
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 100, ptr %1, align 8
// CHECK-NEXT:   %2 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %1, 1
// CHECK-NEXT:   %3 = call i64 @"{{.*}}/cl/_testrt/any.incVal"(%"{{.*}}/runtime/internal/runtime.eface" %2)
// CHECK-NEXT:   %4 = call i32 (ptr, ...) @printf(ptr @3, ptr %0, i64 %3)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequal8"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.memequal8"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequal64"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.memequal64"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }
