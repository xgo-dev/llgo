// LITTEST
package main

import (
	"github.com/goplus/lib/c"
	"github.com/goplus/lib/py"
	"github.com/goplus/lib/py/math"
)

// CHECK-LABEL: define void @"{{.*}}/cl/_testpy/pow.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @PyFloat_FromDouble(double 2.000000e+00)
// CHECK-NEXT:   %1 = call ptr @PyFloat_FromDouble(double 3.000000e+00)
// CHECK-NEXT:   %2 = load ptr, ptr @__llgo_py.math.pow, align 8
// CHECK-NEXT:   %3 = call ptr (ptr, ...) @PyObject_CallFunctionObjArgs(ptr %2, ptr %0, ptr %1, ptr null)
// CHECK-NEXT:   %4 = call double @PyFloat_AsDouble(ptr %3)
// CHECK-NEXT:   %5 = call i32 (ptr, ...) @printf(ptr @0, double %4)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func main() {
	x := math.Pow(py.Float(2), py.Float(3))
	c.Printf(c.Str("pow(2, 3) = %f\n"), x.Float64())
}
