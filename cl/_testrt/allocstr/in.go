// LITTEST
package main

import (
	"github.com/goplus/lib/c"
)

// CHECK-LINE: @0 = private unnamed_addr constant [12 x i8] c"Hello world\0A", align 1

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/cl/_testrt/allocstr.hello"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 12 }
// CHECK-NEXT: }

func hello() string {
	return "Hello world\n"
}

func main() {
	c.Printf(c.AllocaCStr(hello()))
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/allocstr.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testrt/allocstr.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testrt/allocstr.init$guard", align 1
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/allocstr.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/cl/_testrt/allocstr.hello"()
// CHECK-NEXT:   %1 = extractvalue %"{{.*}}/runtime/internal/runtime.String" %0, 1
// CHECK-NEXT:   %2 = add i64 %1, 1
// CHECK-NEXT:   %3 = alloca i8, i64 %2, align 1
// CHECK-NEXT:   %4 = call ptr @"{{.*}}/runtime/internal/runtime.CStrCopy"(ptr %3, %"{{.*}}/runtime/internal/runtime.String" %0)
// CHECK-NEXT:   %5 = call i32 (ptr, ...) @printf(ptr %4)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
