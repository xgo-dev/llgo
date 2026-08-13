// LITTEST
package main

import (
	"github.com/goplus/lib/c"
	"github.com/goplus/lib/c/math/cmplx"
)

// CHECK-LABEL: define void @main.f({ float, float } %0, { float, float } %1, ptr %2){{.*}} {
// CHECK: call void @"{{.*}}PrintPointer"(ptr %2)
// CHECK: [[ABS32:%[0-9]+]] = call float @cabsf({ float, float } %0)
// CHECK: [[ABS64:%[0-9]+]] = fpext float [[ABS32]] to double
// CHECK-NEXT: call void @"{{.*}}PrintFloat"(double [[ABS64]])
// CHECK: [[REAL32:%[0-9]+]] = extractvalue { float, float } %1, 0
// CHECK: [[REAL64:%[0-9]+]] = fpext float [[REAL32]] to double
// CHECK-NEXT: call void @"{{.*}}PrintFloat"(double [[REAL64]])
// CHECK: [[IMAG32:%[0-9]+]] = extractvalue { float, float } %1, 1
// CHECK: [[IMAG64:%[0-9]+]] = fpext float [[IMAG32]] to double
// CHECK-NEXT: call void @"{{.*}}PrintFloat"(double [[IMAG64]])
// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: call void @main.f({ float, float } { float 3.000000e+00, float 4.000000e+00 }, { float, float } { float 3.000000e+00, float 4.000000e+00 }, ptr @main.f)

func f(c, z complex64, addr c.Pointer) {
	println("addr:", addr)
	println("abs(3+4i):", cmplx.Absf(c))
	println("real(3+4i):", real(z))
	println("imag(3+4i):", imag(z))
}

func main() {
	re := float32(3.0)
	im := float32(4.0)
	z := complex64(3 + 4i)
	x := complex(re, im)
	f(x, z, c.Func(f))
}
