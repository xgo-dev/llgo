// LITTEST
package main

import (
	"github.com/goplus/lib/py"
	"github.com/goplus/lib/py/std"
)

// CHECK-LABEL: define void @"{{.*}}/cl/_testpy/max.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @PyFloat_FromDouble(double 3.000000e+00)
// CHECK-NEXT:   %1 = call ptr @PyFloat_FromDouble(double 9.000000e+00)
// CHECK-NEXT:   %2 = call ptr @PyFloat_FromDouble(double 2.300000e+01)
// CHECK-NEXT:   %3 = call ptr @PyFloat_FromDouble(double 1.000000e+02)
// CHECK-NEXT:   %4 = load ptr, ptr @__llgo_py.builtins.max, align 8
// CHECK-NEXT:   %5 = call ptr (ptr, ...) @PyObject_CallFunctionObjArgs(ptr %4, ptr %0, ptr %1, ptr %2, ptr %3, ptr null)
// CHECK-NEXT:   %6 = load ptr, ptr @__llgo_py.builtins.print, align 8
// CHECK-NEXT:   %7 = call ptr (ptr, ...) @PyObject_CallFunctionObjArgs(ptr %6, ptr %5, ptr null)
// CHECK-NEXT:   %8 = call ptr @PyList_New(i64 4)
// CHECK-NEXT:   %9 = call ptr @PyFloat_FromDouble(double 3.000000e+00)
// CHECK-NEXT:   %10 = call i32 @PyList_SetItem(ptr %8, i64 0, ptr %9)
// CHECK-NEXT:   %11 = call ptr @PyFloat_FromDouble(double 9.000000e+00)
// CHECK-NEXT:   %12 = call i32 @PyList_SetItem(ptr %8, i64 1, ptr %11)
// CHECK-NEXT:   %13 = call ptr @PyFloat_FromDouble(double 2.300000e+01)
// CHECK-NEXT:   %14 = call i32 @PyList_SetItem(ptr %8, i64 2, ptr %13)
// CHECK-NEXT:   %15 = call ptr @PyFloat_FromDouble(double 1.000000e+02)
// CHECK-NEXT:   %16 = call i32 @PyList_SetItem(ptr %8, i64 3, ptr %15)
// CHECK-NEXT:   %17 = load ptr, ptr @__llgo_py.builtins.iter, align 8
// CHECK-NEXT:   %18 = call ptr @PyObject_CallOneArg(ptr %17, ptr %8)
// CHECK-NEXT:   %19 = load ptr, ptr @__llgo_py.builtins.max, align 8
// CHECK-NEXT:   %20 = call ptr (ptr, ...) @PyObject_CallFunctionObjArgs(ptr %19, ptr %18, ptr null)
// CHECK-NEXT:   %21 = load ptr, ptr @__llgo_py.builtins.print, align 8
// CHECK-NEXT:   %22 = call ptr (ptr, ...) @PyObject_CallFunctionObjArgs(ptr %21, ptr %20, ptr null)
// CHECK-NEXT:   %23 = call ptr @PyTuple_New(i64 3)
// CHECK-NEXT:   %24 = call ptr @PyFloat_FromDouble(double 1.000000e+00)
// CHECK-NEXT:   %25 = call i32 @PyTuple_SetItem(ptr %23, i64 0, ptr %24)
// CHECK-NEXT:   %26 = call ptr @PyFloat_FromDouble(double 2.000000e+00)
// CHECK-NEXT:   %27 = call i32 @PyTuple_SetItem(ptr %23, i64 1, ptr %26)
// CHECK-NEXT:   %28 = call ptr @PyFloat_FromDouble(double 3.000000e+00)
// CHECK-NEXT:   %29 = call i32 @PyTuple_SetItem(ptr %23, i64 2, ptr %28)
// CHECK-NEXT:   %30 = load ptr, ptr @__llgo_py.builtins.iter, align 8
// CHECK-NEXT:   %31 = call ptr @PyObject_CallOneArg(ptr %30, ptr %23)
// CHECK-NEXT:   %32 = load ptr, ptr @__llgo_py.builtins.max, align 8
// CHECK-NEXT:   %33 = call ptr (ptr, ...) @PyObject_CallFunctionObjArgs(ptr %32, ptr %31, ptr null)
// CHECK-NEXT:   %34 = load ptr, ptr @__llgo_py.builtins.print, align 8
// CHECK-NEXT:   %35 = call ptr (ptr, ...) @PyObject_CallFunctionObjArgs(ptr %34, ptr %33, ptr null)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func main() {
	x := std.Max(py.Float(3.0), py.Float(9.0), py.Float(23.0), py.Float(100.0))
	std.Print(x)

	list := py.List(3.0, 9.0, 23.0, 100.0)
	y := std.Max(std.Iter(list))
	std.Print(y)

	tuple := py.Tuple(1.0, 2.0, 3.0)
	z := std.Max(std.Iter(tuple))
	std.Print(z)
}
