// LITTEST
package main

import (
	"github.com/goplus/lib/c"
	"github.com/goplus/lib/py/math"
)

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: [[MATH:%[0-9]+]] = load ptr, ptr @__llgo_py.math
// CHECK-NEXT: [[PI:%[0-9]+]] = call ptr @PyObject_GetAttrString(ptr [[MATH]], ptr @{{[0-9]+}})
// CHECK-NEXT: [[PI_VALUE:%[0-9]+]] = call double @PyFloat_AsDouble(ptr [[PI]])
// CHECK-NEXT: call i32 (ptr, ...) @printf(ptr @{{[0-9]+}}, double [[PI_VALUE]])
func main() {
	c.Printf(c.Str("pi = %f\n"), math.Pi.Float64())
}
