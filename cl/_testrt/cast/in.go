// LITTEST
package main

//"github.com/goplus/lib/c"

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/cast.cvt32Fto32"(float %0, i32 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = fptosi float %0 to i64
// CHECK-NEXT:   %3 = trunc i64 %2 to i32
// CHECK-NEXT:   %4 = icmp ne i32 %3, %1
// CHECK-NEXT:   br i1 %4, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %5 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 5 }, ptr %5, align 8
// CHECK-NEXT:   %6 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %5, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %6)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func cvt32Fto32(a float32, b int32) {
	if int32(a) != b {
		panic("error")
	}
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/cast.cvt32Fto32U"(float %0, i32 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = fcmp olt float %0, 0.000000e+00
// CHECK-NEXT:   %3 = fptosi float %0 to i64
// CHECK-NEXT:   %4 = fptoui float %0 to i64
// CHECK-NEXT:   %5 = select i1 %2, i64 %3, i64 %4
// CHECK-NEXT:   %6 = trunc i64 %5 to i32
// CHECK-NEXT:   %7 = icmp ne i32 %6, %1
// CHECK-NEXT:   br i1 %7, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %8 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 5 }, ptr %8, align 8
// CHECK-NEXT:   %9 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %8, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %9)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func cvt32Fto32U(a float32, b uint32) {
	if uint32(a) != b {
		panic("error")
	}
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/cast.cvt32Fto64F"(float %0, double %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = fpext float %0 to double
// CHECK-NEXT:   %3 = fcmp une double %2, %1
// CHECK-NEXT:   br i1 %3, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %4 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 5 }, ptr %4, align 8
// CHECK-NEXT:   %5 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %4, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %5)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func cvt32Fto64F(a float32, b float64) {
	if float64(a) != b {
		panic("error")
	}
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/cast.cvt32Fto8"(float %0, i8 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = fptosi float %0 to i64
// CHECK-NEXT:   %3 = trunc i64 %2 to i8
// CHECK-NEXT:   %4 = icmp ne i8 %3, %1
// CHECK-NEXT:   br i1 %4, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %5 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 5 }, ptr %5, align 8
// CHECK-NEXT:   %6 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %5, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %6)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func cvt32Fto8(a float32, b int8) {
	if int8(a) != b {
		panic("error")
	}
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/cast.cvt32Fto8U"(float %0, i8 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = fcmp olt float %0, 0.000000e+00
// CHECK-NEXT:   %3 = fptosi float %0 to i64
// CHECK-NEXT:   %4 = fptoui float %0 to i64
// CHECK-NEXT:   %5 = select i1 %2, i64 %3, i64 %4
// CHECK-NEXT:   %6 = trunc i64 %5 to i8
// CHECK-NEXT:   %7 = icmp ne i8 %6, %1
// CHECK-NEXT:   br i1 %7, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %8 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 5 }, ptr %8, align 8
// CHECK-NEXT:   %9 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %8, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %9)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func cvt32Fto8U(a float32, b uint8) {
	if uint8(a) != b {
		panic("error")
	}
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/cast.cvt32to64"(i32 %0, i64 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = sext i32 %0 to i64
// CHECK-NEXT:   %3 = icmp ne i64 %2, %1
// CHECK-NEXT:   br i1 %3, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %4 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 5 }, ptr %4, align 8
// CHECK-NEXT:   %5 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %4, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %5)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func cvt32to64(a int32, b int64) {
	if int64(a) != b {
		panic("error")
	}
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/cast.cvt64Fto32F"(double %0, float %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = fptrunc double %0 to float
// CHECK-NEXT:   %3 = fcmp une float %2, %1
// CHECK-NEXT:   br i1 %3, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %4 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 5 }, ptr %4, align 8
// CHECK-NEXT:   %5 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %4, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %5)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func cvt64Fto32F(a float64, b float32) {
	if float32(a) != b {
		panic("error")
	}
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/cast.cvt64Uto64F"(i64 %0, double %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = uitofp i64 %0 to double
// CHECK-NEXT:   %3 = fcmp une double %2, %1
// CHECK-NEXT:   br i1 %3, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %4 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 5 }, ptr %4, align 8
// CHECK-NEXT:   %5 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %4, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %5)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func cvt64Uto64F(a uint64, b float64) {
	if float64(a) != b {
		panic("error")
	}
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/cast.cvt64to64F"(i64 %0, double %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = sitofp i64 %0 to double
// CHECK-NEXT:   %3 = fcmp une double %2, %1
// CHECK-NEXT:   br i1 %3, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %4 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 5 }, ptr %4, align 8
// CHECK-NEXT:   %5 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %4, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %5)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func cvt64to64F(a int64, b float64) {
	if float64(a) != b {
		panic("error")
	}
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/cast.cvt64to8"(i64 %0, i8 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = trunc i64 %0 to i8
// CHECK-NEXT:   %3 = icmp ne i8 %2, %1
// CHECK-NEXT:   br i1 %3, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %4 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 5 }, ptr %4, align 8
// CHECK-NEXT:   %5 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %4, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %5)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func cvt64to8(a int64, b int8) {
	if int8(a) != b {
		panic("error")
	}
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/cast.cvt64to8U"(i64 %0, i8 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = trunc i64 %0 to i8
// CHECK-NEXT:   %3 = icmp ne i8 %2, %1
// CHECK-NEXT:   br i1 %3, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %4 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 5 }, ptr %4, align 8
// CHECK-NEXT:   %5 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %4, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %5)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func cvt64to8U(a int, b uint8) {
	if uint8(a) != b {
		panic("error")
	}
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/cast.cvtFtoUintptr"(double %0, i64 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = fptoui double %0 to i64
// CHECK-NEXT:   %3 = icmp ne i64 %2, %1
// CHECK-NEXT:   br i1 %3, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %4 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 5 }, ptr %4, align 8
// CHECK-NEXT:   %5 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %4, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %5)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func cvtFtoUintptr(a float64, b uintptr) {
	if uintptr(a) != b {
		panic("error")
	}
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/cast.cvtUinptr"(i32 %0, i64 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = sext i32 %0 to i64
// CHECK-NEXT:   %3 = icmp ne i64 %2, %1
// CHECK-NEXT:   br i1 %3, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %4 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 5 }, ptr %4, align 8
// CHECK-NEXT:   %5 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %4, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %5)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %6 = trunc i64 %1 to i32
// CHECK-NEXT:   %7 = icmp ne i32 %6, %0
// CHECK-NEXT:   br i1 %7, label %_llgo_3, label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %8 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 5 }, ptr %8, align 8
// CHECK-NEXT:   %9 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %8, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %9)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_2
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func cvtUinptr(a int32, b uintptr) {
	if uintptr(a) != b {
		panic("error")
	}
	if int32(b) != a {
		panic("error")
	}
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/cast.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/cast.cvt64to8"(i64 0, i8 0)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/cast.cvt64to8"(i64 127, i8 127)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/cast.cvt64to8"(i64 128, i8 -128)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/cast.cvt64to8"(i64 -128, i8 -128)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/cast.cvt64to8"(i64 -129, i8 127)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/cast.cvt64to8"(i64 256, i8 0)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/cast.cvt64to8U"(i64 0, i8 0)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/cast.cvt64to8U"(i64 255, i8 -1)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/cast.cvt64to8U"(i64 256, i8 0)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/cast.cvt64to8U"(i64 257, i8 1)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/cast.cvt64to8U"(i64 -1, i8 -1)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/cast.cvt32Fto8"(float 0x3FB99999A0000000, i8 0)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/cast.cvt32Fto8"(float 0x405FC66660000000, i8 127)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/cast.cvt32Fto8"(float 0x4060033340000000, i8 -128)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/cast.cvt32Fto8"(float 0xC060033340000000, i8 -128)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/cast.cvt32Fto8"(float 0xC060233340000000, i8 127)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/cast.cvt32Fto8"(float 0x40700199A0000000, i8 0)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/cast.cvt32Fto8U"(float 0.000000e+00, i8 0)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/cast.cvt32Fto8U"(float 2.550000e+02, i8 -1)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/cast.cvt32Fto8U"(float 2.560000e+02, i8 0)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/cast.cvt32Fto8U"(float 2.570000e+02, i8 1)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/cast.cvt32Fto8U"(float -1.000000e+00, i8 -1)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/cast.cvt32Fto32"(float 0.000000e+00, i32 0)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/cast.cvt32Fto32"(float 1.500000e+00, i32 1)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/cast.cvt32Fto32"(float 0x41D1194D80000000, i32 1147483648)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/cast.cvt32Fto32"(float 0xC1E0000000000000, i32 -2147483648)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/cast.cvt32Fto32U"(float 0.000000e+00, i32 0)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/cast.cvt32Fto32U"(float 1.500000e+00, i32 1)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/cast.cvt32Fto32U"(float 0x41F0000000000000, i32 0)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/cast.cvt32Fto32U"(float 0x41F3B9ACA0000000, i32 1000000000)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/cast.cvt32Fto32U"(float 0xC1F0000000000000, i32 0)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/cast.cvt32Fto32U"(float 0xC1D34BE880000000, i32 -1294967296)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/cast.cvt32Fto32U"(float 0xBFF19999A0000000, i32 -1)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/cast.cvt32Fto64F"(float 0.000000e+00, double 0.000000e+00)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/cast.cvt32Fto64F"(float 1.500000e+00, double 1.500000e+00)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/cast.cvt32Fto64F"(float 1.000000e+10, double 1.000000e+10)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/cast.cvt32Fto64F"(float -1.000000e+10, double -1.000000e+10)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/cast.cvt64Fto32F"(double 0.000000e+00, float 0.000000e+00)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/cast.cvt64Fto32F"(double 1.500000e+00, float 1.500000e+00)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/cast.cvt64Fto32F"(double 1.000000e+10, float 1.000000e+10)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/cast.cvt64Fto32F"(double -1.000000e+10, float -1.000000e+10)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/cast.cvt64to64F"(i64 0, double 0.000000e+00)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/cast.cvt64to64F"(i64 10000000000, double 1.000000e+10)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/cast.cvt64to64F"(i64 9223372036854775807, double 0x43E0000000000000)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/cast.cvt64to64F"(i64 -9223372036854775807, double 0xC3E0000000000000)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/cast.cvt64Uto64F"(i64 0, double 0.000000e+00)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/cast.cvt64Uto64F"(i64 10000000000, double 1.000000e+10)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/cast.cvt64Uto64F"(i64 9223372036854775807, double 0x43E0000000000000)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/cast.cvt64Uto64F"(i64 -1, double 0x43F0000000000000)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/cast.cvt32to64"(i32 0, i64 0)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/cast.cvt32to64"(i32 2147483647, i64 2147483647)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/cast.cvtUinptr"(i32 1024, i64 1024)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/cast.cvtFtoUintptr"(double 1.000000e+02, i64 100)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/cast.cvtFtoUintptr"(double 0.000000e+00, i64 0)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/cast.cvtFtoUintptr"(double 1.000000e+05, i64 100000)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func main() {
	cvt64to8(0, 0)
	cvt64to8(127, 127)
	cvt64to8(128, -128)
	cvt64to8(-128, -128)
	cvt64to8(-129, 127)
	cvt64to8(256, 0)

	cvt64to8U(0, 0)
	cvt64to8U(255, 255)
	cvt64to8U(256, 0)
	cvt64to8U(257, 1)
	cvt64to8U(-1, 255)

	cvt32Fto8(0.1, 0)
	cvt32Fto8(127.1, 127)
	cvt32Fto8(128.1, -128)
	cvt32Fto8(-128.1, -128)
	cvt32Fto8(-129.1, 127)
	cvt32Fto8(256.1, 0)

	cvt32Fto8U(0, 0)
	cvt32Fto8U(255, 255)
	cvt32Fto8U(256, 0)
	cvt32Fto8U(257, 1)
	cvt32Fto8U(-1, 255)

	// MaxInt32  = 1<<31 - 1           // 2147483647
	// MinInt32  = -1 << 31            // -2147483648
	cvt32Fto32(0, 0)
	cvt32Fto32(1.5, 1)
	cvt32Fto32(1147483647.1, 1147483648)
	cvt32Fto32(-2147483648.1, -2147483648)

	// MaxUint32 = 1<<32 - 1           // 4294967295
	cvt32Fto32U(0, 0)
	cvt32Fto32U(1.5, 1)
	cvt32Fto32U(4294967295.1, 0)
	cvt32Fto32U(5294967295.1, 1000000000)
	cvt32Fto32U(-4294967295.1, 0)
	cvt32Fto32U(-1294967295.1, 3000000000)
	cvt32Fto32U(-1.1, 4294967295)

	// MaxFloat32             = 0x1p127 * (1 + (1 - 0x1p-23)) // 3.40282346638528859811704183484516925440e+38
	// SmallestNonzeroFloat32 = 0x1p-126 * 0x1p-23            // 1.401298464324817070923729583289916131280e-45
	// MaxFloat64             = 0x1p1023 * (1 + (1 - 0x1p-52)) // 1.79769313486231570814527423731704356798070e+308
	// SmallestNonzeroFloat64 = 0x1p-1022 * 0x1p-52            // 4.9406564584124654417656879286822137236505980e-324

	cvt32Fto64F(0, 0)
	cvt32Fto64F(1.5, 1.5)
	cvt32Fto64F(1e10, 1e10)
	cvt32Fto64F(-1e10, -1e10)

	cvt64Fto32F(0, 0)
	cvt64Fto32F(1.5, 1.5)
	cvt64Fto32F(1e10, 1e10)
	cvt64Fto32F(-1e10, -1e10)

	// MaxInt64  = 1<<63 - 1           // 9223372036854775807
	// MinInt64  = -1 << 63            // -9223372036854775808
	cvt64to64F(0, 0)
	cvt64to64F(1e10, 1e10)
	cvt64to64F(9223372036854775807, 9223372036854775807)
	cvt64to64F(-9223372036854775807, -9223372036854775807)

	// MaxUint64 = 1<<64 - 1           // 18446744073709551615
	cvt64Uto64F(0, 0)
	cvt64Uto64F(1e10, 1e10)
	cvt64Uto64F(9223372036854775807, 9223372036854775807)
	cvt64Uto64F(18446744073709551615, 18446744073709551615)

	cvt32to64(0, 0)
	cvt32to64(2147483647, 2147483647)

	cvtUinptr(1024, 1024)

	cvtFtoUintptr(100.0, 100)
	cvtFtoUintptr(0.0, 0)
	cvtFtoUintptr(1e5, 100000)
}
