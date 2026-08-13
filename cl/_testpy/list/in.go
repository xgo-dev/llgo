// LITTEST
package main

import (
	"unsafe"

	"github.com/goplus/lib/c"
	"github.com/goplus/lib/py"
	"github.com/goplus/lib/py/math"
	"github.com/goplus/lib/py/std"
)

func main() {
	v := 100
	x := py.List(true, false, 1, float32(2.1), 3.1, uint(4), 1+2i, complex64(3+4i),
		"hello", []byte("world"), [...]byte{1, 2, 3}, [...]byte{}, &v, unsafe.Pointer(&v))
	y := py.List(std.Abs, std.Print, math.Pi)
	c.Printf(c.Str("lens = %d %d\n"), x.ListLen(), y.ListLen())
	c.Printf(c.Str("ptrs = %d %d\n"), x.ListItem(12).IsTrue(), x.ListItem(13).IsTrue())
	c.Printf(c.Str("pi = %.15g\n"), y.ListItem(2).Float64())
}

// Python module symbol binding belongs to package initialization, not main's
// list construction.
// CHECK-LABEL: define void @main.init(){{.*}} {
// CHECK: [[BUILTINS:%[0-9]+]] = load ptr, ptr @__llgo_py.builtins
// CHECK-NEXT: call void (ptr, ...) @llgoLoadPyModSyms(ptr [[BUILTINS]], ptr @{{[0-9]+}}, ptr @__llgo_py.builtins.abs, ptr @{{[0-9]+}}, ptr @__llgo_py.builtins.print, ptr null)

// CHECK-LABEL: define void @main.main(){{.*}} {
// Each Go value must be converted to the corresponding Python object and
// inserted into the same 14-element list at its source position.
// CHECK: [[GO_PTR:%[0-9]+]] = call ptr @"{{.*}}AllocZ"(i64 8)
// CHECK: [[WORLD:%[0-9]+]] = call %"{{.*}}Slice" @"{{.*}}StringToBytes"(%"{{.*}}String" { ptr @{{[0-9]+}}, i64 5 })
// CHECK: [[X:%[0-9]+]] = call ptr @PyList_New(i64 14)
// CHECK-NEXT: [[X0:%[0-9]+]] = call ptr @PyBool_FromLong(i32 -1)
// CHECK-NEXT: call i32 @PyList_SetItem(ptr [[X]], i64 0, ptr [[X0]])
// CHECK-NEXT: [[X1:%[0-9]+]] = call ptr @PyBool_FromLong(i32 0)
// CHECK-NEXT: call i32 @PyList_SetItem(ptr [[X]], i64 1, ptr [[X1]])
// CHECK-NEXT: [[X2:%[0-9]+]] = call ptr @PyLong_FromLongLong(i64 1)
// CHECK-NEXT: call i32 @PyList_SetItem(ptr [[X]], i64 2, ptr [[X2]])
// CHECK-NEXT: [[X3:%[0-9]+]] = call ptr @PyFloat_FromDouble(double 0x4000CCCCC0000000)
// CHECK-NEXT: call i32 @PyList_SetItem(ptr [[X]], i64 3, ptr [[X3]])
// CHECK-NEXT: [[X4:%[0-9]+]] = call ptr @PyFloat_FromDouble(double 3.100000e+00)
// CHECK-NEXT: call i32 @PyList_SetItem(ptr [[X]], i64 4, ptr [[X4]])
// CHECK-NEXT: [[X5:%[0-9]+]] = call ptr @PyLong_FromUnsignedLongLong(i64 4)
// CHECK-NEXT: call i32 @PyList_SetItem(ptr [[X]], i64 5, ptr [[X5]])
// CHECK-NEXT: [[X6:%[0-9]+]] = call ptr @PyComplex_FromDoubles(double 1.000000e+00, double 2.000000e+00)
// CHECK-NEXT: call i32 @PyList_SetItem(ptr [[X]], i64 6, ptr [[X6]])
// CHECK-NEXT: [[X7:%[0-9]+]] = call ptr @PyComplex_FromDoubles(double 3.000000e+00, double 4.000000e+00)
// CHECK-NEXT: call i32 @PyList_SetItem(ptr [[X]], i64 7, ptr [[X7]])
// CHECK-NEXT: [[X8:%[0-9]+]] = call ptr @PyUnicode_FromStringAndSize(ptr @{{[0-9]+}}, i64 5)
// CHECK-NEXT: call i32 @PyList_SetItem(ptr [[X]], i64 8, ptr [[X8]])
// CHECK-NEXT: [[WORLD_DATA:%[0-9]+]] = extractvalue %"{{.*}}Slice" [[WORLD]], 0
// CHECK-NEXT: [[WORLD_LEN:%[0-9]+]] = extractvalue %"{{.*}}Slice" [[WORLD]], 1
// CHECK-NEXT: [[X9:%[0-9]+]] = call ptr @PyByteArray_FromStringAndSize(ptr [[WORLD_DATA]], i64 [[WORLD_LEN]])
// CHECK-NEXT: call i32 @PyList_SetItem(ptr [[X]], i64 9, ptr [[X9]])
// CHECK: [[ARRAY_DATA:%[0-9]+]] = getelementptr inbounds ptr, ptr {{.*}}, i64 0
// CHECK-NEXT: [[X10:%[0-9]+]] = call ptr @PyBytes_FromStringAndSize(ptr [[ARRAY_DATA]], i64 3)
// CHECK-NEXT: call i32 @PyList_SetItem(ptr [[X]], i64 10, ptr [[X10]])
// CHECK-NEXT: [[X11:%[0-9]+]] = call ptr @PyBytes_FromStringAndSize(ptr null, i64 0)
// CHECK-NEXT: call i32 @PyList_SetItem(ptr [[X]], i64 11, ptr [[X11]])
// CHECK-NEXT: [[PTR_INT0:%[0-9]+]] = ptrtoint ptr [[GO_PTR]] to i64
// CHECK-NEXT: [[X12:%[0-9]+]] = call ptr @PyLong_FromUnsignedLongLong(i64 [[PTR_INT0]])
// CHECK-NEXT: call i32 @PyList_SetItem(ptr [[X]], i64 12, ptr [[X12]])
// CHECK-NEXT: [[PTR_INT1:%[0-9]+]] = ptrtoint ptr [[GO_PTR]] to i64
// CHECK-NEXT: [[X13:%[0-9]+]] = call ptr @PyLong_FromUnsignedLongLong(i64 [[PTR_INT1]])
// CHECK-NEXT: call i32 @PyList_SetItem(ptr [[X]], i64 13, ptr [[X13]])
// The second list contains the two loaded builtins and math.Pi, and all later
// observations must read back from the correct list and index.
// CHECK: [[MATH_MOD:%[0-9]+]] = load ptr, ptr @__llgo_py.math
// CHECK-NEXT: [[PI:%[0-9]+]] = call ptr @PyObject_GetAttrString(ptr [[MATH_MOD]], ptr @{{[0-9]+}})
// CHECK-NEXT: [[Y:%[0-9]+]] = call ptr @PyList_New(i64 3)
// CHECK-NEXT: [[ABS:%[0-9]+]] = load ptr, ptr @__llgo_py.builtins.abs
// CHECK-NEXT: call i32 @PyList_SetItem(ptr [[Y]], i64 0, ptr [[ABS]])
// CHECK-NEXT: [[PRINT:%[0-9]+]] = load ptr, ptr @__llgo_py.builtins.print
// CHECK-NEXT: call i32 @PyList_SetItem(ptr [[Y]], i64 1, ptr [[PRINT]])
// CHECK-NEXT: call i32 @PyList_SetItem(ptr [[Y]], i64 2, ptr [[PI]])
// CHECK-NEXT: [[XLEN:%[0-9]+]] = call i64 @PyList_Size(ptr [[X]])
// CHECK-NEXT: [[YLEN:%[0-9]+]] = call i64 @PyList_Size(ptr [[Y]])
// CHECK-NEXT: call i32 (ptr, ...) @printf(ptr @{{[0-9]+}}, i64 [[XLEN]], i64 [[YLEN]])
// CHECK-NEXT: [[PTR0:%[0-9]+]] = call ptr @PyList_GetItem(ptr [[X]], i64 12)
// CHECK-NEXT: [[PTR0_TRUE:%[0-9]+]] = call i32 @PyObject_IsTrue(ptr [[PTR0]])
// CHECK-NEXT: [[PTR1:%[0-9]+]] = call ptr @PyList_GetItem(ptr [[X]], i64 13)
// CHECK-NEXT: [[PTR1_TRUE:%[0-9]+]] = call i32 @PyObject_IsTrue(ptr [[PTR1]])
// CHECK-NEXT: call i32 (ptr, ...) @printf(ptr @{{[0-9]+}}, i32 [[PTR0_TRUE]], i32 [[PTR1_TRUE]])
// CHECK-NEXT: [[PI_ITEM:%[0-9]+]] = call ptr @PyList_GetItem(ptr [[Y]], i64 2)
// CHECK-NEXT: [[PI_VALUE:%[0-9]+]] = call double @PyFloat_AsDouble(ptr [[PI_ITEM]])
// CHECK-NEXT: call i32 (ptr, ...) @printf(ptr @{{[0-9]+}}, double [[PI_VALUE]])
