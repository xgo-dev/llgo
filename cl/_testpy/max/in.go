// LITTEST
package main

import (
	"github.com/goplus/lib/py"
	"github.com/goplus/lib/py/std"
)

// CHECK-LABEL: define void @main.main(){{.*}} {
// Direct variadic Max passes the four converted objects to the loaded builtin,
// and Print consumes the returned object.
// CHECK: [[D0:%[0-9]+]] = call ptr @PyFloat_FromDouble(double 3.000000e+00)
// CHECK-NEXT: [[D1:%[0-9]+]] = call ptr @PyFloat_FromDouble(double 9.000000e+00)
// CHECK-NEXT: [[D2:%[0-9]+]] = call ptr @PyFloat_FromDouble(double 2.300000e+01)
// CHECK-NEXT: [[D3:%[0-9]+]] = call ptr @PyFloat_FromDouble(double 1.000000e+02)
// CHECK-NEXT: [[MAX_FN:%[0-9]+]] = load ptr, ptr @__llgo_py.builtins.max
// CHECK-NEXT: [[DIRECT_MAX:%[0-9]+]] = call ptr (ptr, ...) @PyObject_CallFunctionObjArgs(ptr [[MAX_FN]], ptr [[D0]], ptr [[D1]], ptr [[D2]], ptr [[D3]], ptr null)
// CHECK-NEXT: [[PRINT0:%[0-9]+]] = load ptr, ptr @__llgo_py.builtins.print
// CHECK-NEXT: call ptr (ptr, ...) @PyObject_CallFunctionObjArgs(ptr [[PRINT0]], ptr [[DIRECT_MAX]], ptr null)
// CHECK: [[LIST:%[0-9]+]] = call ptr @PyList_New(i64 4)
// CHECK-NEXT: [[L0:%[0-9]+]] = call ptr @PyFloat_FromDouble(double 3.000000e+00)
// CHECK-NEXT: call i32 @PyList_SetItem(ptr [[LIST]], i64 0, ptr [[L0]])
// CHECK-NEXT: [[L1:%[0-9]+]] = call ptr @PyFloat_FromDouble(double 9.000000e+00)
// CHECK-NEXT: call i32 @PyList_SetItem(ptr [[LIST]], i64 1, ptr [[L1]])
// CHECK-NEXT: [[L2:%[0-9]+]] = call ptr @PyFloat_FromDouble(double 2.300000e+01)
// CHECK-NEXT: call i32 @PyList_SetItem(ptr [[LIST]], i64 2, ptr [[L2]])
// CHECK-NEXT: [[L3:%[0-9]+]] = call ptr @PyFloat_FromDouble(double 1.000000e+02)
// CHECK-NEXT: call i32 @PyList_SetItem(ptr [[LIST]], i64 3, ptr [[L3]])
// CHECK-NEXT: [[ITER_FN:%[0-9]+]] = load ptr, ptr @__llgo_py.builtins.iter
// CHECK-NEXT: [[LIST_ITER:%[0-9]+]] = call ptr @PyObject_CallOneArg(ptr [[ITER_FN]], ptr [[LIST]])
// CHECK-NEXT: [[MAX1:%[0-9]+]] = load ptr, ptr @__llgo_py.builtins.max
// CHECK-NEXT: [[LIST_MAX:%[0-9]+]] = call ptr (ptr, ...) @PyObject_CallFunctionObjArgs(ptr [[MAX1]], ptr [[LIST_ITER]], ptr null)
// CHECK-NEXT: [[PRINT1:%[0-9]+]] = load ptr, ptr @__llgo_py.builtins.print
// CHECK-NEXT: call ptr (ptr, ...) @PyObject_CallFunctionObjArgs(ptr [[PRINT1]], ptr [[LIST_MAX]], ptr null)
// CHECK: [[TUPLE:%[0-9]+]] = call ptr @PyTuple_New(i64 3)
// CHECK-NEXT: [[T0:%[0-9]+]] = call ptr @PyFloat_FromDouble(double 1.000000e+00)
// CHECK-NEXT: call i32 @PyTuple_SetItem(ptr [[TUPLE]], i64 0, ptr [[T0]])
// CHECK-NEXT: [[T1:%[0-9]+]] = call ptr @PyFloat_FromDouble(double 2.000000e+00)
// CHECK-NEXT: call i32 @PyTuple_SetItem(ptr [[TUPLE]], i64 1, ptr [[T1]])
// CHECK-NEXT: [[T2:%[0-9]+]] = call ptr @PyFloat_FromDouble(double 3.000000e+00)
// CHECK-NEXT: call i32 @PyTuple_SetItem(ptr [[TUPLE]], i64 2, ptr [[T2]])
// CHECK-NEXT: [[ITER_FN2:%[0-9]+]] = load ptr, ptr @__llgo_py.builtins.iter
// CHECK-NEXT: [[TUPLE_ITER:%[0-9]+]] = call ptr @PyObject_CallOneArg(ptr [[ITER_FN2]], ptr [[TUPLE]])
// CHECK-NEXT: [[MAX2:%[0-9]+]] = load ptr, ptr @__llgo_py.builtins.max
// CHECK-NEXT: [[TUPLE_MAX:%[0-9]+]] = call ptr (ptr, ...) @PyObject_CallFunctionObjArgs(ptr [[MAX2]], ptr [[TUPLE_ITER]], ptr null)
// CHECK-NEXT: [[PRINT2:%[0-9]+]] = load ptr, ptr @__llgo_py.builtins.print
// CHECK-NEXT: call ptr (ptr, ...) @PyObject_CallFunctionObjArgs(ptr [[PRINT2]], ptr [[TUPLE_MAX]], ptr null)
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
