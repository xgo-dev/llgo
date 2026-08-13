// LITTEST
package math

import (
	_ "unsafe"

	"github.com/goplus/lib/py"
)

// CHECK-LABEL: define void @"{{.*}}/cl/_testpy/math.init"(){{.*}} {
// CHECK: [[GUARD:%[0-9]+]] = load i1, ptr @"{{.*}}/cl/_testpy/math.init$guard"
// CHECK: store i1 true, ptr @"{{.*}}/cl/_testpy/math.init$guard"
// CHECK-NEXT: [[OLD_MATH:%[0-9]+]] = load ptr, ptr @__llgo_py.math
// CHECK-NEXT: [[HAS_MATH:%[0-9]+]] = icmp ne ptr [[OLD_MATH]], null
// CHECK: [[MATH:%[0-9]+]] = call ptr @PyImport_ImportModule(ptr @{{[0-9]+}})
// CHECK-NEXT: store ptr [[MATH]], ptr @__llgo_py.math

const (
	LLGoPackage = "py.math"
)

//go:linkname Sqrt py.sqrt
func Sqrt(x *py.Object) *py.Object
