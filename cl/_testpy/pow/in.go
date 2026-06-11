// LITTEST
package main

import (
	"github.com/goplus/lib/c"
	"github.com/goplus/lib/py"
	"github.com/goplus/lib/py/math"
)

// CHECK-LINE: @0 = private unnamed_addr constant [16 x i8] c"pow(2, 3) = %f\0A\00", align 1
// CHECK-LINE: @1 = private unnamed_addr constant [4 x i8] c"pow\00", align 1

func main() {
	x := math.Pow(py.Float(2), py.Float(3))
	c.Printf(c.Str("pow(2, 3) = %f\n"), x.Float64())
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testpy/pow.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testpy/pow.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testpy/pow.init$guard", align 1
// CHECK-NEXT:   call void @"github.com/goplus/lib/py/math.init"()
// CHECK-NEXT:   %1 = load ptr, ptr @__llgo_py.math, align 8
// CHECK-NEXT:   call void (ptr, ...) @llgoLoadPyModSyms(ptr %1, ptr @1, ptr @__llgo_py.math.pow, ptr null)
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testpy/pow.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @PyFloat_FromDouble(double 2.000000e+00)
// CHECK-NEXT:   %1 = call ptr @PyFloat_FromDouble(double 3.000000e+00)
// CHECK-NEXT:   %2 = load ptr, ptr @__llgo_py.math.pow, align 8
// CHECK-NEXT:   %3 = call ptr (ptr, ...) @PyObject_CallFunctionObjArgs(ptr %2, ptr %0, ptr %1, ptr null)
// CHECK-NEXT:   %4 = call double @PyFloat_AsDouble(ptr %3)
// CHECK-NEXT:   %5 = call i32 (ptr, ...) @printf(ptr @0, double %4)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
