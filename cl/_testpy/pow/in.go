// LITTEST
package main

import (
	"github.com/goplus/lib/c"
	"github.com/goplus/lib/py"
	"github.com/goplus/lib/py/math"
)

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: [[BASE:%[0-9]+]] = call ptr @PyFloat_FromDouble(double 2.000000e+00)
// CHECK-NEXT: [[EXP:%[0-9]+]] = call ptr @PyFloat_FromDouble(double 3.000000e+00)
// CHECK-NEXT: [[POW_FN:%[0-9]+]] = load ptr, ptr @__llgo_py.math.pow
// CHECK-NEXT: [[POW:%[0-9]+]] = call ptr (ptr, ...) @PyObject_CallFunctionObjArgs(ptr [[POW_FN]], ptr [[BASE]], ptr [[EXP]], ptr null)
// CHECK-NEXT: [[POW_VALUE:%[0-9]+]] = call double @PyFloat_AsDouble(ptr [[POW]])
// CHECK-NEXT: call i32 (ptr, ...) @printf(ptr @{{[0-9]+}}, double [[POW_VALUE]])
func main() {
	x := math.Pow(py.Float(2), py.Float(3))
	c.Printf(c.Str("pow(2, 3) = %f\n"), x.Float64())
}
