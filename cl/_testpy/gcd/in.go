// LITTEST
package main

import (
	"github.com/goplus/lib/c"
	"github.com/goplus/lib/py"
	"github.com/goplus/lib/py/math"
)

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: [[G0:%[0-9]+]] = call ptr @PyLong_FromLong(i64 60)
// CHECK-NEXT: [[G1:%[0-9]+]] = call ptr @PyLong_FromLong(i64 20)
// CHECK-NEXT: [[G2:%[0-9]+]] = call ptr @PyLong_FromLong(i64 25)
// CHECK-NEXT: [[GCD_FN:%[0-9]+]] = load ptr, ptr @__llgo_py.math.gcd
// CHECK-NEXT: [[GCD:%[0-9]+]] = call ptr (ptr, ...) @PyObject_CallFunctionObjArgs(ptr [[GCD_FN]], ptr [[G0]], ptr [[G1]], ptr [[G2]], ptr null)
// CHECK-NEXT: [[GCD_VALUE:%[0-9]+]] = call i64 @PyLong_AsLong(ptr [[GCD]])
// CHECK-NEXT: call i32 (ptr, ...) @printf(ptr @{{[0-9]+}}, i64 [[GCD_VALUE]])
func main() {
	x := math.Gcd(py.Long(60), py.Long(20), py.Long(25))
	c.Printf(c.Str("gcd(60, 20, 25) = %d\n"), x.Long())
}
