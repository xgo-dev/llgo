// LITTEST
package main

import (
	"github.com/goplus/lib/c"
	"github.com/goplus/lib/py/math"
)

// CHECK-LABEL: define void @"{{.*}}/cl/_testpy/pi.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load ptr, ptr @__llgo_py.math, align 8
// CHECK-NEXT:   %1 = call ptr @PyObject_GetAttrString(ptr %0, ptr @1)
// CHECK-NEXT:   %2 = call double @PyFloat_AsDouble(ptr %1)
// CHECK-NEXT:   %3 = call i32 (ptr, ...) @printf(ptr @0, double %2)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func main() {
	c.Printf(c.Str("pi = %f\n"), math.Pi.Float64())
}
