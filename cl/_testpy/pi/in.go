// LITTEST
package main

import (
	"github.com/goplus/lib/c"
	"github.com/goplus/lib/py/math"
)

// CHECK-LINE: @0 = private unnamed_addr constant [9 x i8] c"pi = %f\0A\00", align 1
// CHECK-LINE: @1 = private unnamed_addr constant [3 x i8] c"pi\00", align 1

func main() {
	c.Printf(c.Str("pi = %f\n"), math.Pi.Float64())
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testpy/pi.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testpy/pi.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testpy/pi.init$guard", align 1
// CHECK-NEXT:   call void @"github.com/goplus/lib/py/math.init"()
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testpy/pi.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load ptr, ptr @__llgo_py.math, align 8
// CHECK-NEXT:   %1 = call ptr @PyObject_GetAttrString(ptr %0, ptr @1)
// CHECK-NEXT:   %2 = call double @PyFloat_AsDouble(ptr %1)
// CHECK-NEXT:   %3 = call i32 (ptr, ...) @printf(ptr @0, double %2)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
