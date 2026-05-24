// LITTEST
package main

import (
	"github.com/goplus/lib/c"
	"github.com/goplus/lib/py"
	"github.com/goplus/lib/py/math"
)

// CHECK-LINE: @0 = private unnamed_addr constant [22 x i8] c"gcd(60, 20, 25) = %d\0A\00", align 1
// CHECK-LINE: @1 = private unnamed_addr constant [4 x i8] c"gcd\00", align 1

func main() {
	x := math.Gcd(py.Long(60), py.Long(20), py.Long(25))
	c.Printf(c.Str("gcd(60, 20, 25) = %d\n"), x.Long())
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testpy/gcd.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testpy/gcd.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testpy/gcd.init$guard", align 1
// CHECK-NEXT:   call void @"github.com/goplus/lib/py/math.init"()
// CHECK-NEXT:   %1 = load ptr, ptr @__llgo_py.math, align 8
// CHECK-NEXT:   call void (ptr, ...) @llgoLoadPyModSyms(ptr %1, ptr @1, ptr @__llgo_py.math.gcd, ptr null)
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

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
