// LITTEST
package main

import (
	"github.com/goplus/lib/c"
	"github.com/goplus/lib/py"
	"github.com/goplus/lib/py/math"
	"github.com/goplus/lib/py/os"
	"github.com/goplus/lib/py/std"
)

// CHECK-LABEL: define void @"{{.*}}/cl/_testpy/callpy.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testpy/callpy.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testpy/callpy.init$guard", align 1
// CHECK-NEXT:   call void @"github.com/goplus/lib/py/math.init"()
// CHECK-NEXT:   call void @"github.com/goplus/lib/py/os.init"()
// CHECK-NEXT:   call void @"github.com/goplus/lib/py/std.init"()
// CHECK-NEXT:   %1 = load ptr, ptr @__llgo_py.builtins, align 8
// CHECK-NEXT:   call void (ptr, ...) @llgoLoadPyModSyms(ptr %1, ptr @2, ptr @__llgo_py.builtins.print, ptr null)
// CHECK-NEXT:   %2 = load ptr, ptr @__llgo_py.math, align 8
// CHECK-NEXT:   call void (ptr, ...) @llgoLoadPyModSyms(ptr %2, ptr @3, ptr @__llgo_py.math.sqrt, ptr null)
// CHECK-NEXT:   %3 = load ptr, ptr @__llgo_py.os, align 8
// CHECK-NEXT:   call void (ptr, ...) @llgoLoadPyModSyms(ptr %3, ptr @4, ptr @__llgo_py.os.getcwd, ptr null)
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
// CHECK-NEXT:   %7 = call ptr @PyUnicode_FromString(ptr @1)
// CHECK-NEXT:   %8 = load ptr, ptr @__llgo_py.builtins.print, align 8
// CHECK-NEXT:   %9 = call ptr (ptr, ...) @PyObject_CallFunctionObjArgs(ptr %8, ptr %7, ptr %4, ptr null)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func main() {
	x := math.Sqrt(py.Float(2))
	wd := os.Getcwd()
	c.Printf(c.Str("sqrt(2) = %f\n"), x.Float64())
	std.Print(py.Str("cwd ="), wd)
}
