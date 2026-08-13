// LITTEST
package main

import (
	"github.com/goplus/lib/c"
	"github.com/goplus/lib/py"
	"github.com/goplus/lib/py/numpy"
)

// CHECK-LABEL: define void @main.main(){{.*}} {
// Build each row from its own scalar objects, then insert those rows into the
// corresponding outer matrix. This prevents a flat sequence of SetItem calls
// from accidentally satisfying the test with crossed lists or indices.
// CHECK: [[A0:%[0-9]+]] = call ptr @PyList_New(i64 3)
// CHECK-NEXT: [[A00:%[0-9]+]] = call ptr @PyFloat_FromDouble(double 1.000000e+00)
// CHECK-NEXT: call i32 @PyList_SetItem(ptr [[A0]], i64 0, ptr [[A00]])
// CHECK-NEXT: [[A01:%[0-9]+]] = call ptr @PyFloat_FromDouble(double 2.000000e+00)
// CHECK-NEXT: call i32 @PyList_SetItem(ptr [[A0]], i64 1, ptr [[A01]])
// CHECK-NEXT: [[A02:%[0-9]+]] = call ptr @PyFloat_FromDouble(double 3.000000e+00)
// CHECK-NEXT: call i32 @PyList_SetItem(ptr [[A0]], i64 2, ptr [[A02]])
// CHECK-NEXT: [[A1:%[0-9]+]] = call ptr @PyList_New(i64 3)
// CHECK-NEXT: [[A10:%[0-9]+]] = call ptr @PyFloat_FromDouble(double 4.000000e+00)
// CHECK-NEXT: call i32 @PyList_SetItem(ptr [[A1]], i64 0, ptr [[A10]])
// CHECK-NEXT: [[A11:%[0-9]+]] = call ptr @PyFloat_FromDouble(double 5.000000e+00)
// CHECK-NEXT: call i32 @PyList_SetItem(ptr [[A1]], i64 1, ptr [[A11]])
// CHECK-NEXT: [[A12:%[0-9]+]] = call ptr @PyFloat_FromDouble(double 6.000000e+00)
// CHECK-NEXT: call i32 @PyList_SetItem(ptr [[A1]], i64 2, ptr [[A12]])
// CHECK-NEXT: [[A2:%[0-9]+]] = call ptr @PyList_New(i64 3)
// CHECK-NEXT: [[A20:%[0-9]+]] = call ptr @PyFloat_FromDouble(double 7.000000e+00)
// CHECK-NEXT: call i32 @PyList_SetItem(ptr [[A2]], i64 0, ptr [[A20]])
// CHECK-NEXT: [[A21:%[0-9]+]] = call ptr @PyFloat_FromDouble(double 8.000000e+00)
// CHECK-NEXT: call i32 @PyList_SetItem(ptr [[A2]], i64 1, ptr [[A21]])
// CHECK-NEXT: [[A22:%[0-9]+]] = call ptr @PyFloat_FromDouble(double 9.000000e+00)
// CHECK-NEXT: call i32 @PyList_SetItem(ptr [[A2]], i64 2, ptr [[A22]])
// CHECK-NEXT: [[A:%[0-9]+]] = call ptr @PyList_New(i64 3)
// CHECK-NEXT: call i32 @PyList_SetItem(ptr [[A]], i64 0, ptr [[A0]])
// CHECK-NEXT: call i32 @PyList_SetItem(ptr [[A]], i64 1, ptr [[A1]])
// CHECK-NEXT: call i32 @PyList_SetItem(ptr [[A]], i64 2, ptr [[A2]])
// CHECK-NEXT: [[B0:%[0-9]+]] = call ptr @PyList_New(i64 3)
// CHECK-NEXT: [[B00:%[0-9]+]] = call ptr @PyFloat_FromDouble(double 9.000000e+00)
// CHECK-NEXT: call i32 @PyList_SetItem(ptr [[B0]], i64 0, ptr [[B00]])
// CHECK-NEXT: [[B01:%[0-9]+]] = call ptr @PyFloat_FromDouble(double 8.000000e+00)
// CHECK-NEXT: call i32 @PyList_SetItem(ptr [[B0]], i64 1, ptr [[B01]])
// CHECK-NEXT: [[B02:%[0-9]+]] = call ptr @PyFloat_FromDouble(double 7.000000e+00)
// CHECK-NEXT: call i32 @PyList_SetItem(ptr [[B0]], i64 2, ptr [[B02]])
// CHECK-NEXT: [[B1:%[0-9]+]] = call ptr @PyList_New(i64 3)
// CHECK-NEXT: [[B10:%[0-9]+]] = call ptr @PyFloat_FromDouble(double 6.000000e+00)
// CHECK-NEXT: call i32 @PyList_SetItem(ptr [[B1]], i64 0, ptr [[B10]])
// CHECK-NEXT: [[B11:%[0-9]+]] = call ptr @PyFloat_FromDouble(double 5.000000e+00)
// CHECK-NEXT: call i32 @PyList_SetItem(ptr [[B1]], i64 1, ptr [[B11]])
// CHECK-NEXT: [[B12:%[0-9]+]] = call ptr @PyFloat_FromDouble(double 4.000000e+00)
// CHECK-NEXT: call i32 @PyList_SetItem(ptr [[B1]], i64 2, ptr [[B12]])
// CHECK-NEXT: [[B2:%[0-9]+]] = call ptr @PyList_New(i64 3)
// CHECK-NEXT: [[B20:%[0-9]+]] = call ptr @PyFloat_FromDouble(double 3.000000e+00)
// CHECK-NEXT: call i32 @PyList_SetItem(ptr [[B2]], i64 0, ptr [[B20]])
// CHECK-NEXT: [[B21:%[0-9]+]] = call ptr @PyFloat_FromDouble(double 2.000000e+00)
// CHECK-NEXT: call i32 @PyList_SetItem(ptr [[B2]], i64 1, ptr [[B21]])
// CHECK-NEXT: [[B22:%[0-9]+]] = call ptr @PyFloat_FromDouble(double 1.000000e+00)
// CHECK-NEXT: call i32 @PyList_SetItem(ptr [[B2]], i64 2, ptr [[B22]])
// CHECK-NEXT: [[B:%[0-9]+]] = call ptr @PyList_New(i64 3)
// CHECK-NEXT: call i32 @PyList_SetItem(ptr [[B]], i64 0, ptr [[B0]])
// CHECK-NEXT: call i32 @PyList_SetItem(ptr [[B]], i64 1, ptr [[B1]])
// CHECK-NEXT: call i32 @PyList_SetItem(ptr [[B]], i64 2, ptr [[B2]])
// CHECK-NEXT: [[ADD_FN:%[0-9]+]] = load ptr, ptr @__llgo_py.numpy.add
// CHECK-NEXT: [[SUM:%[0-9]+]] = call ptr (ptr, ...) @PyObject_CallFunctionObjArgs(ptr [[ADD_FN]], ptr [[A]], ptr [[B]], ptr null)
// Each printed C string must derive from the corresponding matrix/result.
// CHECK-NEXT: [[A_REPR:%[0-9]+]] = call ptr @PyObject_Str(ptr [[A]])
// CHECK-NEXT: [[A_CSTR:%[0-9]+]] = call ptr @PyUnicode_AsUTF8(ptr [[A_REPR]])
// CHECK-NEXT: call i32 (ptr, ...) @printf(ptr @{{[0-9]+}}, ptr [[A_CSTR]])
// CHECK-NEXT: [[B_REPR:%[0-9]+]] = call ptr @PyObject_Str(ptr [[B]])
// CHECK-NEXT: [[B_CSTR:%[0-9]+]] = call ptr @PyUnicode_AsUTF8(ptr [[B_REPR]])
// CHECK-NEXT: call i32 (ptr, ...) @printf(ptr @{{[0-9]+}}, ptr [[B_CSTR]])
// CHECK-NEXT: [[SUM_REPR:%[0-9]+]] = call ptr @PyObject_Str(ptr [[SUM]])
// CHECK-NEXT: [[SUM_CSTR:%[0-9]+]] = call ptr @PyUnicode_AsUTF8(ptr [[SUM_REPR]])
// CHECK-NEXT: call i32 (ptr, ...) @printf(ptr @{{[0-9]+}}, ptr [[SUM_CSTR]])
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
