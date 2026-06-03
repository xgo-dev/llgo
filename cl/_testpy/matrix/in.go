// LITTEST
package main

import (
	"github.com/goplus/lib/c"
	"github.com/goplus/lib/py"
	"github.com/goplus/lib/py/numpy"
)

// CHECK-LABEL: define void @"{{.*}}/cl/_testpy/matrix.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @PyList_New(i64 3)
// CHECK-NEXT:   %1 = call ptr @PyFloat_FromDouble(double 1.000000e+00)
// CHECK-NEXT:   %2 = call i32 @PyList_SetItem(ptr %0, i64 0, ptr %1)
// CHECK-NEXT:   %3 = call ptr @PyFloat_FromDouble(double 2.000000e+00)
// CHECK-NEXT:   %4 = call i32 @PyList_SetItem(ptr %0, i64 1, ptr %3)
// CHECK-NEXT:   %5 = call ptr @PyFloat_FromDouble(double 3.000000e+00)
// CHECK-NEXT:   %6 = call i32 @PyList_SetItem(ptr %0, i64 2, ptr %5)
// CHECK-NEXT:   %7 = call ptr @PyList_New(i64 3)
// CHECK-NEXT:   %8 = call ptr @PyFloat_FromDouble(double 4.000000e+00)
// CHECK-NEXT:   %9 = call i32 @PyList_SetItem(ptr %7, i64 0, ptr %8)
// CHECK-NEXT:   %10 = call ptr @PyFloat_FromDouble(double 5.000000e+00)
// CHECK-NEXT:   %11 = call i32 @PyList_SetItem(ptr %7, i64 1, ptr %10)
// CHECK-NEXT:   %12 = call ptr @PyFloat_FromDouble(double 6.000000e+00)
// CHECK-NEXT:   %13 = call i32 @PyList_SetItem(ptr %7, i64 2, ptr %12)
// CHECK-NEXT:   %14 = call ptr @PyList_New(i64 3)
// CHECK-NEXT:   %15 = call ptr @PyFloat_FromDouble(double 7.000000e+00)
// CHECK-NEXT:   %16 = call i32 @PyList_SetItem(ptr %14, i64 0, ptr %15)
// CHECK-NEXT:   %17 = call ptr @PyFloat_FromDouble(double 8.000000e+00)
// CHECK-NEXT:   %18 = call i32 @PyList_SetItem(ptr %14, i64 1, ptr %17)
// CHECK-NEXT:   %19 = call ptr @PyFloat_FromDouble(double 9.000000e+00)
// CHECK-NEXT:   %20 = call i32 @PyList_SetItem(ptr %14, i64 2, ptr %19)
// CHECK-NEXT:   %21 = call ptr @PyList_New(i64 3)
// CHECK-NEXT:   %22 = call i32 @PyList_SetItem(ptr %21, i64 0, ptr %0)
// CHECK-NEXT:   %23 = call i32 @PyList_SetItem(ptr %21, i64 1, ptr %7)
// CHECK-NEXT:   %24 = call i32 @PyList_SetItem(ptr %21, i64 2, ptr %14)
// CHECK-NEXT:   %25 = call ptr @PyList_New(i64 3)
// CHECK-NEXT:   %26 = call ptr @PyFloat_FromDouble(double 9.000000e+00)
// CHECK-NEXT:   %27 = call i32 @PyList_SetItem(ptr %25, i64 0, ptr %26)
// CHECK-NEXT:   %28 = call ptr @PyFloat_FromDouble(double 8.000000e+00)
// CHECK-NEXT:   %29 = call i32 @PyList_SetItem(ptr %25, i64 1, ptr %28)
// CHECK-NEXT:   %30 = call ptr @PyFloat_FromDouble(double 7.000000e+00)
// CHECK-NEXT:   %31 = call i32 @PyList_SetItem(ptr %25, i64 2, ptr %30)
// CHECK-NEXT:   %32 = call ptr @PyList_New(i64 3)
// CHECK-NEXT:   %33 = call ptr @PyFloat_FromDouble(double 6.000000e+00)
// CHECK-NEXT:   %34 = call i32 @PyList_SetItem(ptr %32, i64 0, ptr %33)
// CHECK-NEXT:   %35 = call ptr @PyFloat_FromDouble(double 5.000000e+00)
// CHECK-NEXT:   %36 = call i32 @PyList_SetItem(ptr %32, i64 1, ptr %35)
// CHECK-NEXT:   %37 = call ptr @PyFloat_FromDouble(double 4.000000e+00)
// CHECK-NEXT:   %38 = call i32 @PyList_SetItem(ptr %32, i64 2, ptr %37)
// CHECK-NEXT:   %39 = call ptr @PyList_New(i64 3)
// CHECK-NEXT:   %40 = call ptr @PyFloat_FromDouble(double 3.000000e+00)
// CHECK-NEXT:   %41 = call i32 @PyList_SetItem(ptr %39, i64 0, ptr %40)
// CHECK-NEXT:   %42 = call ptr @PyFloat_FromDouble(double 2.000000e+00)
// CHECK-NEXT:   %43 = call i32 @PyList_SetItem(ptr %39, i64 1, ptr %42)
// CHECK-NEXT:   %44 = call ptr @PyFloat_FromDouble(double 1.000000e+00)
// CHECK-NEXT:   %45 = call i32 @PyList_SetItem(ptr %39, i64 2, ptr %44)
// CHECK-NEXT:   %46 = call ptr @PyList_New(i64 3)
// CHECK-NEXT:   %47 = call i32 @PyList_SetItem(ptr %46, i64 0, ptr %25)
// CHECK-NEXT:   %48 = call i32 @PyList_SetItem(ptr %46, i64 1, ptr %32)
// CHECK-NEXT:   %49 = call i32 @PyList_SetItem(ptr %46, i64 2, ptr %39)
// CHECK-NEXT:   %50 = load ptr, ptr @__llgo_py.numpy.add, align 8
// CHECK-NEXT:   %51 = call ptr (ptr, ...) @PyObject_CallFunctionObjArgs(ptr %50, ptr %21, ptr %46, ptr null)
// CHECK-NEXT:   %52 = call ptr @PyObject_Str(ptr %21)
// CHECK-NEXT:   %53 = call ptr @PyUnicode_AsUTF8(ptr %52)
// CHECK-NEXT:   %54 = call i32 (ptr, ...) @printf(ptr @0, ptr %53)
// CHECK-NEXT:   %55 = call ptr @PyObject_Str(ptr %46)
// CHECK-NEXT:   %56 = call ptr @PyUnicode_AsUTF8(ptr %55)
// CHECK-NEXT:   %57 = call i32 (ptr, ...) @printf(ptr @1, ptr %56)
// CHECK-NEXT:   %58 = call ptr @PyObject_Str(ptr %51)
// CHECK-NEXT:   %59 = call ptr @PyUnicode_AsUTF8(ptr %58)
// CHECK-NEXT:   %60 = call i32 (ptr, ...) @printf(ptr @2, ptr %59)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func main() {
	a := py.List(
		py.List(1.0, 2.0, 3.0),
		py.List(4.0, 5.0, 6.0),
		py.List(7.0, 8.0, 9.0),
	)
	b := py.List(
		py.List(9.0, 8.0, 7.0),
		py.List(6.0, 5.0, 4.0),
		py.List(3.0, 2.0, 1.0),
	)
	x := numpy.Add(a, b)
	c.Printf(c.Str("a = %s\n"), a.Str().CStr())
	c.Printf(c.Str("a = %s\n"), b.Str().CStr())
	c.Printf(c.Str("a+b = %s\n"), x.Str().CStr())
}
