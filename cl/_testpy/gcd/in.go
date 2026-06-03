// LITTEST
package main

import (
	"github.com/goplus/lib/c"
	"github.com/goplus/lib/py"
	"github.com/goplus/lib/py/math"
)

// CHECK-LABEL: define void @"{{.*}}/cl/_testpy/gcd.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @PyLong_FromLong(i64 60)
// CHECK-NEXT:   %1 = call ptr @PyLong_FromLong(i64 20)
// CHECK-NEXT:   %2 = call ptr @PyLong_FromLong(i64 25)
// CHECK-NEXT:   %3 = load ptr, ptr @__llgo_py.math.gcd, align 8
// CHECK-NEXT:   %4 = call ptr (ptr, ...) @PyObject_CallFunctionObjArgs(ptr %3, ptr %0, ptr %1, ptr %2, ptr null)
// CHECK-NEXT:   %5 = call i64 @PyLong_AsLong(ptr %4)
// CHECK-NEXT:   %6 = call i32 (ptr, ...) @printf(ptr @0, i64 %5)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func main() {
	x := math.Gcd(py.Long(60), py.Long(20), py.Long(25))
	c.Printf(c.Str("gcd(60, 20, 25) = %d\n"), x.Long())
}
