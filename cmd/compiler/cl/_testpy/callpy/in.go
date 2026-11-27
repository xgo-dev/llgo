// LITTEST
// Scope: os+arch (Python C long width; call forms otherwise common)
package main

import (
	"github.com/xgo-dev/llgo/cmd/compiler/cl/_testpy/callpy/builtins"
	_ "github.com/xgo-dev/llgo/cmd/compiler/cl/_testpy/callpy/pylink"
)

// This fixture owns Python zero-, one-, two-, and variadic-argument calls plus
// boolean, float, and platform-long result extraction. The imported packages
// are deliberately tiny local ABI declarations, not wrapper-library coverage.
func main() {
	truth := builtins.Bool()
	converted := builtins.Float(builtins.NewFloat(2))
	power := builtins.Pow(builtins.NewFloat(2), builtins.NewFloat(3))
	integer := builtins.Int(builtins.NewFloat(5))
	maximum := builtins.Max(
		builtins.NewFloat(3),
		builtins.NewFloat(9),
		builtins.NewFloat(23),
		builtins.NewFloat(100),
	)
	maximumAgain := builtins.Max(builtins.NewFloat(-1), builtins.NewFloat(4))

	println("bool =", truth.IsTrue())
	println("float =", converted.Float64() == 2)
	println("pow =", power.Float64() == 8)
	println("long =", integer.Long())
	println("max =", maximum.Float64() == 100)
	println("max again =", maximumAgain.Float64() == 4)
}

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: [[BOOL_FN:%[0-9]+]] = load ptr, ptr @__llgo_py.builtins.bool
// CHECK: [[BOOL:%[0-9]+]] = call ptr @PyObject_CallNoArgs(ptr [[BOOL_FN]])
// CHECK: [[FLOAT_ARG:%[0-9]+]] = call ptr @PyFloat_FromDouble(double 2.000000e+00)
// CHECK: [[FLOAT_FN:%[0-9]+]] = load ptr, ptr @__llgo_py.builtins.float
// CHECK: [[FLOAT:%[0-9]+]] = call ptr @PyObject_CallOneArg(ptr [[FLOAT_FN]], ptr [[FLOAT_ARG]])
// CHECK: [[POW_FN:%[0-9]+]] = load ptr, ptr @__llgo_py.builtins.pow
// CHECK: [[POWER:%[0-9]+]] = call ptr (ptr, ...) @PyObject_CallFunctionObjArgs(ptr [[POW_FN]], ptr %{{[0-9]+}}, ptr %{{[0-9]+}}, ptr null)
// CHECK: [[INT_FN:%[0-9]+]] = load ptr, ptr @__llgo_py.builtins.int
// CHECK: [[INTEGER:%[0-9]+]] = call ptr @PyObject_CallOneArg(ptr [[INT_FN]], ptr %{{[0-9]+}})
// CHECK: [[MAX_FN:%[0-9]+]] = load ptr, ptr @__llgo_py.builtins.max
// CHECK: [[MAX:%[0-9]+]] = call ptr (ptr, ...) @PyObject_CallFunctionObjArgs(ptr [[MAX_FN]], ptr %{{[0-9]+}}, ptr %{{[0-9]+}}, ptr %{{[0-9]+}}, ptr %{{[0-9]+}}, ptr null)
// The same pyfunc must remain callable after the first invocation (funcOf
// historically cached/destructively transformed this slot).
// CHECK: [[MAX_FN_AGAIN:%[0-9]+]] = load ptr, ptr @__llgo_py.builtins.max
// CHECK: [[MAX_AGAIN:%[0-9]+]] = call ptr (ptr, ...) @PyObject_CallFunctionObjArgs(ptr [[MAX_FN_AGAIN]], ptr %{{[0-9]+}}, ptr %{{[0-9]+}}, ptr null)
// CHECK: call i32 @PyObject_IsTrue(ptr [[BOOL]])
// CHECK: call double @PyFloat_AsDouble(ptr [[FLOAT]])
// CHECK: call double @PyFloat_AsDouble(ptr [[POWER]])
// CHECK: call i{{32|64}} @PyLong_AsLong(ptr [[INTEGER]])
// CHECK: call double @PyFloat_AsDouble(ptr [[MAX]])
// CHECK: call double @PyFloat_AsDouble(ptr [[MAX_AGAIN]])
// WINDOWS: declare i32 @PyLong_AsLong(ptr)
