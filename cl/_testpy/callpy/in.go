// LITTEST
package main

import (
	"github.com/goplus/lib/c"
	"github.com/goplus/lib/py"
	"github.com/goplus/lib/py/math"
	"github.com/goplus/lib/py/os"
)

func main() {
	x := math.Sqrt(py.Float(2))
	wd := os.Getcwd()
	c.Printf(c.Str("sqrt(2) = %.6f\n"), x.Float64())
	c.Printf(c.Str("cwd ok = %d\n"), int(wd.IsTrue()))
}

// CHECK-LABEL: define void @main.init(){{.*}} {
// CHECK: [[MATH_MOD:%[0-9]+]] = load ptr, ptr @__llgo_py.math
// CHECK-NEXT: call void (ptr, ...) @llgoLoadPyModSyms(ptr [[MATH_MOD]], ptr @{{[0-9]+}}, ptr @__llgo_py.math.sqrt, ptr null)
// CHECK-NEXT: [[OS_MOD:%[0-9]+]] = load ptr, ptr @__llgo_py.os
// CHECK-NEXT: call void (ptr, ...) @llgoLoadPyModSyms(ptr [[OS_MOD]], ptr @{{[0-9]+}}, ptr @__llgo_py.os.getcwd, ptr null)

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: [[TWO:%[0-9]+]] = call ptr @PyFloat_FromDouble(double 2.000000e+00)
// CHECK-NEXT: [[SQRT_FN:%[0-9]+]] = load ptr, ptr @__llgo_py.math.sqrt
// CHECK-NEXT: [[SQRT:%[0-9]+]] = call ptr @PyObject_CallOneArg(ptr [[SQRT_FN]], ptr [[TWO]])
// CHECK-NEXT: [[GETCWD_FN:%[0-9]+]] = load ptr, ptr @__llgo_py.os.getcwd
// CHECK-NEXT: [[CWD:%[0-9]+]] = call ptr @PyObject_CallNoArgs(ptr [[GETCWD_FN]])
// CHECK-NEXT: [[SQRT_VALUE:%[0-9]+]] = call double @PyFloat_AsDouble(ptr [[SQRT]])
// CHECK-NEXT: call i32 (ptr, ...) @printf(ptr @{{[0-9]+}}, double [[SQRT_VALUE]])
// CHECK-NEXT: [[CWD_TRUE:%[0-9]+]] = call i32 @PyObject_IsTrue(ptr [[CWD]])
// CHECK-NEXT: [[CWD_INT:%[0-9]+]] = sext i32 [[CWD_TRUE]] to i64
// CHECK-NEXT: call i32 (ptr, ...) @printf(ptr @{{[0-9]+}}, i64 [[CWD_INT]])
