// LITTEST
package main

import (
	"github.com/goplus/lib/c"
)

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
// CHECK-NEXT:   %4 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @1, i64 58 }, ptr %4, align 8
// CHECK-NEXT:   %5 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %4, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %5)
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
// CHECK-NEXT:   %6 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @4, i64 32 }, ptr %6, align 8
// CHECK-NEXT:   %7 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %6, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %7)
// CHECK-NEXT:   unreachable
// CHECK-NEXT: }
func incVal(a any) int {
	return a.(int) + 1
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/any.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/cl/_testrt/any.hi"(%"{{.*}}/runtime/internal/runtime.eface" { ptr @"*_llgo_int8", ptr @6 })
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 100, ptr %1, align 8
// CHECK-NEXT:   %2 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %1, 1
// CHECK-NEXT:   %3 = call i64 @"{{.*}}/cl/_testrt/any.incVal"(%"{{.*}}/runtime/internal/runtime.eface" %2)
// CHECK-NEXT:   %4 = call i32 (ptr, ...) @printf(ptr @5, ptr %0, i64 %3)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func main() {
	c.Printf(c.Str("%s %d\n"), hi(c.Str("Hello")), incVal(100))
}
