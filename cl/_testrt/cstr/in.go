// LITTEST
package main

import _ "unsafe"

//
//go:linkname cstr llgo.cstr
// CHECK-LINE: @0 = private unnamed_addr constant [14 x i8] c"Hello, world\0A\00", align 1

func cstr(string) *int8

//go:linkname printf C.printf
func printf(format *int8, __llgo_va_list ...any)

func main() {
	printf(cstr("Hello, world\n"))
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/cstr.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testrt/cstr.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testrt/cstr.init$guard", align 1
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/cstr.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   call void (ptr, ...) @printf(ptr @0)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
