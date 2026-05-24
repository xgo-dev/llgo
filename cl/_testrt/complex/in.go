// LITTEST
package main

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/complex.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testrt/complex.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testrt/complex.init$guard", align 1
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

type T complex64

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/complex.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintFloat"(double 1.000000e+00)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintFloat"(double 2.000000e+00)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintComplex"({ double, double } { double -1.000000e+00, double -2.000000e+00 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintComplex"({ double, double } { double 4.000000e+00, double 6.000000e+00 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintComplex"({ double, double } { double -2.000000e+00, double -2.000000e+00 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintComplex"({ double, double } { double -5.000000e+00, double 1.000000e+01 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %0 = call { double, double } @"{{.*}}/runtime/internal/runtime.Complex128Div"({ double, double } { double 1.000000e+00, double 2.000000e+00 }, { double, double } { double 3.000000e+00, double 4.000000e+00 })
// CHECK-NEXT:   %1 = extractvalue { double, double } %0, 0
// CHECK-NEXT:   %2 = extractvalue { double, double } %0, 1
// CHECK-NEXT:   %3 = insertvalue { double, double } undef, double %1, 0
// CHECK-NEXT:   %4 = insertvalue { double, double } %3, double %2, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintComplex"({ double, double } %4)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %5 = call { double, double } @"{{.*}}/runtime/internal/runtime.Complex128Div"({ double, double } { double 1.000000e+00, double 2.000000e+00 }, { double, double } zeroinitializer)
// CHECK-NEXT:   %6 = extractvalue { double, double } %5, 0
// CHECK-NEXT:   %7 = extractvalue { double, double } %5, 1
// CHECK-NEXT:   %8 = insertvalue { double, double } undef, double %6, 0
// CHECK-NEXT:   %9 = insertvalue { double, double } %8, double %7, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintComplex"({ double, double } %9)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %10 = call { double, double } @"{{.*}}/runtime/internal/runtime.Complex128Div"({ double, double } zeroinitializer, { double, double } zeroinitializer)
// CHECK-NEXT:   %11 = extractvalue { double, double } %10, 0
// CHECK-NEXT:   %12 = extractvalue { double, double } %10, 1
// CHECK-NEXT:   %13 = insertvalue { double, double } undef, double %11, 0
// CHECK-NEXT:   %14 = insertvalue { double, double } %13, double %12, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintComplex"({ double, double } %14)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintBool"(i1 true)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintBool"(i1 false)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintBool"(i1 false)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintBool"(i1 true)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintBool"(i1 true)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

func main() {
	a := 1 + 2i
	b := 3 + 4i
	c := 0 + 0i
	println(real(a), imag(a))
	println(-a)
	println(a + b)
	println(a - b)
	println(a * b)
	println(a / b)
	println(a / c)
	println(c / c)
	println(a == a, a != a)
	println(a == b, a != b)
	println(complex128(T(a)) == a)
}
