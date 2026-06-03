// LITTEST
package math

import (
	_ "unsafe"

	"github.com/goplus/lib/py"
)

// CHECK-LABEL: define void @"{{.*}}/cl/_testpy/math.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testpy/math.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testpy/math.init$guard", align 1
// CHECK-NEXT:   %1 = load ptr, ptr @__llgo_py.math, align 8
// CHECK-NEXT:   %2 = icmp ne ptr %1, null
// CHECK-NEXT:   br i1 %2, label %_llgo_2, label %_llgo_3
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_3, %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_1
// CHECK-NEXT:   %3 = call ptr @PyImport_ImportModule(ptr @0)
// CHECK-NEXT:   store ptr %3, ptr @__llgo_py.math, align 8
// CHECK-NEXT:   br label %_llgo_2
// CHECK-NEXT: }

const (
	LLGoPackage = "py.math"
)

//go:linkname Sqrt py.sqrt
func Sqrt(x *py.Object) *py.Object
