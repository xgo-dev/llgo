// LITTEST
package main

// CHECK-LABEL: define void @main.main(){{.*}} {
// Constant arithmetic remains folded, while division is delegated to the
// runtime and its two components are reassembled for printing.
// CHECK: call void @"{{.*}}PrintComplex"({ double, double } { double -1.000000e+00, double -2.000000e+00 })
// CHECK: call void @"{{.*}}PrintComplex"({ double, double } { double 4.000000e+00, double 6.000000e+00 })
// CHECK: call void @"{{.*}}PrintComplex"({ double, double } { double -2.000000e+00, double -2.000000e+00 })
// CHECK: call void @"{{.*}}PrintComplex"({ double, double } { double -5.000000e+00, double 1.000000e+01 })
// CHECK: [[DIV0:%[0-9]+]] = call { double, double } @"{{.*}}Complex128Div"({ double, double } { double 1.000000e+00, double 2.000000e+00 }, { double, double } { double 3.000000e+00, double 4.000000e+00 })
// CHECK-NEXT: [[DIV0_RE:%[0-9]+]] = extractvalue { double, double } [[DIV0]], 0
// CHECK-NEXT: [[DIV0_IM:%[0-9]+]] = extractvalue { double, double } [[DIV0]], 1
// CHECK-NEXT: [[DIV0_PAIR0:%[0-9]+]] = insertvalue { double, double } undef, double [[DIV0_RE]], 0
// CHECK-NEXT: [[DIV0_PAIR:%[0-9]+]] = insertvalue { double, double } [[DIV0_PAIR0]], double [[DIV0_IM]], 1
// CHECK-NEXT: call void @"{{.*}}PrintComplex"({ double, double } [[DIV0_PAIR]])
// CHECK: [[DIV_ZERO:%[0-9]+]] = call { double, double } @"{{.*}}Complex128Div"({ double, double } { double 1.000000e+00, double 2.000000e+00 }, { double, double } zeroinitializer)
// CHECK-NEXT: [[DIV_ZERO_RE:%[0-9]+]] = extractvalue { double, double } [[DIV_ZERO]], 0
// CHECK-NEXT: [[DIV_ZERO_IM:%[0-9]+]] = extractvalue { double, double } [[DIV_ZERO]], 1
// CHECK-NEXT: [[DIV_ZERO_PAIR0:%[0-9]+]] = insertvalue { double, double } undef, double [[DIV_ZERO_RE]], 0
// CHECK-NEXT: [[DIV_ZERO_PAIR:%[0-9]+]] = insertvalue { double, double } [[DIV_ZERO_PAIR0]], double [[DIV_ZERO_IM]], 1
// CHECK-NEXT: call void @"{{.*}}PrintComplex"({ double, double } [[DIV_ZERO_PAIR]])
// CHECK: [[ZERO_DIV_ZERO:%[0-9]+]] = call { double, double } @"{{.*}}Complex128Div"({ double, double } zeroinitializer, { double, double } zeroinitializer)
// CHECK-NEXT: [[ZERO_RE:%[0-9]+]] = extractvalue { double, double } [[ZERO_DIV_ZERO]], 0
// CHECK-NEXT: [[ZERO_IM:%[0-9]+]] = extractvalue { double, double } [[ZERO_DIV_ZERO]], 1
// CHECK-NEXT: [[ZERO_PAIR0:%[0-9]+]] = insertvalue { double, double } undef, double [[ZERO_RE]], 0
// CHECK-NEXT: [[ZERO_PAIR:%[0-9]+]] = insertvalue { double, double } [[ZERO_PAIR0]], double [[ZERO_IM]], 1
// CHECK-NEXT: call void @"{{.*}}PrintComplex"({ double, double } [[ZERO_PAIR]])
// CHECK: call void @"{{.*}}PrintBool"(i1 true)
// CHECK: call void @"{{.*}}PrintBool"(i1 false)
// CHECK: call void @"{{.*}}PrintBool"(i1 false)
// CHECK: call void @"{{.*}}PrintBool"(i1 true)
// CHECK: call void @"{{.*}}PrintBool"(i1 true)
type T complex64

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
