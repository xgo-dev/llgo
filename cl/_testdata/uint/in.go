// LITTEST
package main

import "github.com/goplus/lib/c"

// CHECK-LINE: @0 = private unnamed_addr constant [11 x i8] c"Hello, %u\0A\00", align 1

// CHECK-LABEL: define i32 @"{{.*}}/cl/_testdata/uint.f"(i32 %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = add i32 %0, 1
// CHECK-NEXT:   ret i32 %1
// CHECK-NEXT: }

func f(a c.Uint) c.Uint {
	a++
	return a
}

func main() {
	var a c.Uint = 100
	c.Printf(c.Str("Hello, %u\n"), f(a))
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testdata/uint.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testdata/uint.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testdata/uint.init$guard", align 1
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testdata/uint.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call i32 @"{{.*}}/cl/_testdata/uint.f"(i32 100)
// CHECK-NEXT:   %1 = call i32 (ptr, ...) @printf(ptr @0, i32 %0)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
