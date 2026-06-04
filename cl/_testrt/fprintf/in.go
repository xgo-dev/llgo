// LITTEST
package main

import "unsafe"

//go:linkname cstr llgo.cstr
// CHECK-LINE: @0 = private unnamed_addr constant [10 x i8] c"Hello %d\0A\00", align 1

func cstr(string) *int8

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/fprintf.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testrt/fprintf.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testrt/fprintf.init$guard", align 1
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

//go:linkname stderr __stderrp
var stderr unsafe.Pointer

//go:linkname fprintf C.fprintf
func fprintf(fp unsafe.Pointer, format *int8, __llgo_va_list ...any)

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/fprintf.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load ptr, ptr @__stderrp, align 8
// CHECK-NEXT:   call void (ptr, ptr, ...) @fprintf(ptr %0, ptr @0, i64 100)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

func main() {
	fprintf(stderr, cstr("Hello %d\n"), 100)
}
