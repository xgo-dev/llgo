// LITTEST
package main

import (
	"github.com/goplus/lib/c"
	"github.com/goplus/lib/py"
	"github.com/goplus/lib/py/math"
	"github.com/goplus/lib/py/os"
)

// CHECK-LINE: @0 = private unnamed_addr constant [14 x i8] c"sqrt(2) = %f\0A\00", align 1
// CHECK-LINE: @1 = private unnamed_addr constant [13 x i8] c"cwd ok = %d\0A\00", align 1
// CHECK-LINE: @2 = private unnamed_addr constant [5 x i8] c"sqrt\00", align 1
// CHECK-LINE: @3 = private unnamed_addr constant [7 x i8] c"getcwd\00", align 1

func main() {
	x := math.Sqrt(py.Float(2))
	wd := os.Getcwd()
	c.Printf(c.Str("sqrt(2) = %f\n"), x.Float64())
	ok := 0
	if wd != nil {
		ok = 1
	}
	c.Printf(c.Str("cwd ok = %d\n"), ok)
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testpy/callpy.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testpy/callpy.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testpy/callpy.init$guard", align 1
// CHECK-NEXT:   call void @"github.com/goplus/lib/py/math.init"()
// CHECK-NEXT:   call void @"github.com/goplus/lib/py/os.init"()
// CHECK-NEXT:   %1 = load ptr, ptr @__llgo_py.math, align 8
// CHECK-NEXT:   call void (ptr, ...) @llgoLoadPyModSyms(ptr %1, ptr @2, ptr @__llgo_py.math.sqrt, ptr null)
// CHECK-NEXT:   %2 = load ptr, ptr @__llgo_py.os, align 8
// CHECK-NEXT:   call void (ptr, ...) @llgoLoadPyModSyms(ptr %2, ptr @3, ptr @__llgo_py.os.getcwd, ptr null)
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testpy/callpy.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @PyFloat_FromDouble(double 2.000000e+00)
// CHECK-NEXT:   %1 = load ptr, ptr @__llgo_py.math.sqrt, align 8
// CHECK-NEXT:   %2 = call ptr @PyObject_CallOneArg(ptr %1, ptr %0)
// CHECK-NEXT:   %3 = load ptr, ptr @__llgo_py.os.getcwd, align 8
// CHECK-NEXT:   %4 = call ptr @PyObject_CallNoArgs(ptr %3)
// CHECK-NEXT:   %5 = call double @PyFloat_AsDouble(ptr %2)
// CHECK-NEXT:   %6 = call i32 (ptr, ...) @printf(ptr @0, double %5)
// CHECK-NEXT:   %7 = icmp ne ptr %4, null
// CHECK-NEXT:   br i1 %7, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   %8 = phi i64 [ 0, %_llgo_0 ], [ 1, %_llgo_1 ]
// CHECK-NEXT:   %9 = call i32 (ptr, ...) @printf(ptr @1, i64 %8)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
