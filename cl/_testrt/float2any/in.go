// LITTEST
package main

import "unsafe"

const pi = 3.14159265
const pi32bits = 0x40490fdb
const pi64lo = 0x53c8d4f1
const pi64hi = 0x400921fb

type eface struct {
	typ  unsafe.Pointer
	data unsafe.Pointer
}

type u64parts struct {
	lo uint32
	hi uint32
}

// ESCAPE-LABEL: define void @main.check32(%"{{.*}}/runtime/internal/runtime.eface" %0){{.*}} {
// ESCAPE-NEXT: _llgo_0:
// ESCAPE-NEXT:   %.stack = alloca i8, i64 16, align 8
// ESCAPE-NEXT:   call void @llvm.memset.p0.i64(ptr %.stack, i8 0, i64 16, i1 false)
// ESCAPE-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %0, ptr %.stack, align 8
// ESCAPE-NEXT:   %1 = load %"{{.*}}/runtime/internal/runtime.eface", ptr %.stack, align 8
// ESCAPE-NEXT:   %2 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %1, 0
// ESCAPE-NEXT:   %3 = icmp eq ptr %2, @_llgo_float32
// ESCAPE-NEXT:   br i1 %3, label %_llgo_5, label %_llgo_6
// ESCAPE-EMPTY:
// ESCAPE-NEXT: _llgo_1:                                          ; preds = %_llgo_7
// ESCAPE-NEXT:   %4 = alloca %main.eface, align 8
// ESCAPE-NEXT:   call void @llvm.memset.p0.i64(ptr %4, i8 0, i64 16, i1 false)
// ESCAPE-NEXT:   %5 = load %main.eface, ptr %.stack, align 8
// ESCAPE-NEXT:   store %main.eface %5, ptr %4, align 8
// ESCAPE-NEXT:   %6 = getelementptr inbounds %main.eface, ptr %4, i32 0, i32 1
// ESCAPE-NEXT:   %7 = load ptr, ptr %6, align 8
// ESCAPE-NEXT:   %8 = load i32, ptr %7, align 4
// ESCAPE-NEXT:   %9 = icmp ne i32 %8, 1078530011
// ESCAPE-NEXT:   br i1 %9, label %_llgo_3, label %_llgo_4
// ESCAPE-EMPTY:
// ESCAPE-NEXT: _llgo_2:                                          ; preds = %_llgo_7
// ESCAPE-NEXT:   %10 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// ESCAPE-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @1, i64 14 }, ptr %10, align 8
// ESCAPE-NEXT:   %11 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %10, 1
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %11)
// ESCAPE-NEXT:   unreachable
// ESCAPE-EMPTY:
// ESCAPE-NEXT: _llgo_3:                                          ; preds = %_llgo_1
// ESCAPE-NEXT:   %12 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// ESCAPE-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @3, i64 14 }, ptr %12, align 8
// ESCAPE-NEXT:   %13 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %12, 1
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %13)
// ESCAPE-NEXT:   unreachable
// ESCAPE-EMPTY:
// ESCAPE-NEXT: _llgo_4:                                          ; preds = %_llgo_1
// ESCAPE-NEXT:   ret void
// ESCAPE-EMPTY:
// ESCAPE-NEXT: _llgo_5:                                          ; preds = %_llgo_0
// ESCAPE-NEXT:   %14 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %1, 1
// ESCAPE-NEXT:   %15 = load float, ptr %14, align 4
// ESCAPE-NEXT:   %16 = insertvalue { float, i1 } undef, float %15, 0
// ESCAPE-NEXT:   %17 = insertvalue { float, i1 } %16, i1 true, 1
// ESCAPE-NEXT:   br label %_llgo_7
// ESCAPE-EMPTY:
// ESCAPE-NEXT: _llgo_6:                                          ; preds = %_llgo_0
// ESCAPE-NEXT:   br label %_llgo_7
// ESCAPE-EMPTY:
// ESCAPE-NEXT: _llgo_7:                                          ; preds = %_llgo_6, %_llgo_5
// ESCAPE-NEXT:   %18 = phi { float, i1 } [ %17, %_llgo_5 ], [ zeroinitializer, %_llgo_6 ]
// ESCAPE-NEXT:   %19 = extractvalue { float, i1 } %18, 0
// ESCAPE-NEXT:   %20 = extractvalue { float, i1 } %18, 1
// ESCAPE-NEXT:   br i1 %20, label %_llgo_1, label %_llgo_2
// ESCAPE-NEXT: }

// CHECK-LABEL: define void @main.check32(%"{{.*}}/runtime/internal/runtime.eface" %0){{.*}} {
// CHECK: icmp eq ptr {{.*}}, @_llgo_float32
// CHECK: load i32, ptr
// CHECK: icmp ne i32 {{.*}}, 1078530011
func check32(v any) {
	switch v.(type) {
	case float32:
	default:
		panic("error type f32")
	}
	e := *(*eface)(unsafe.Pointer(&v))
	if *(*uint32)(e.data) != pi32bits {
		panic("error bits f32")
	}
}

// ESCAPE-LABEL: define void @main.check64(%"{{.*}}/runtime/internal/runtime.eface" %0){{.*}} {
// ESCAPE-NEXT: _llgo_0:
// ESCAPE-NEXT:   %.stack = alloca i8, i64 16, align 8
// ESCAPE-NEXT:   call void @llvm.memset.p0.i64(ptr %.stack, i8 0, i64 16, i1 false)
// ESCAPE-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %0, ptr %.stack, align 8
// ESCAPE-NEXT:   %1 = load %"{{.*}}/runtime/internal/runtime.eface", ptr %.stack, align 8
// ESCAPE-NEXT:   %2 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %1, 0
// ESCAPE-NEXT:   %3 = icmp eq ptr %2, @_llgo_float64
// ESCAPE-NEXT:   br i1 %3, label %_llgo_6, label %_llgo_7
// ESCAPE-EMPTY:
// ESCAPE-NEXT: _llgo_1:                                          ; preds = %_llgo_8
// ESCAPE-NEXT:   %4 = alloca %main.eface, align 8
// ESCAPE-NEXT:   call void @llvm.memset.p0.i64(ptr %4, i8 0, i64 16, i1 false)
// ESCAPE-NEXT:   %5 = load %main.eface, ptr %.stack, align 8
// ESCAPE-NEXT:   store %main.eface %5, ptr %4, align 8
// ESCAPE-NEXT:   %6 = alloca %main.u64parts, align 8
// ESCAPE-NEXT:   call void @llvm.memset.p0.i64(ptr %6, i8 0, i64 8, i1 false)
// ESCAPE-NEXT:   %7 = getelementptr inbounds %main.eface, ptr %4, i32 0, i32 1
// ESCAPE-NEXT:   %8 = load ptr, ptr %7, align 8
// ESCAPE-NEXT:   %9 = load %main.u64parts, ptr %8, align 4
// ESCAPE-NEXT:   store %main.u64parts %9, ptr %6, align 4
// ESCAPE-NEXT:   %10 = getelementptr inbounds %main.u64parts, ptr %6, i32 0, i32 0
// ESCAPE-NEXT:   %11 = load i32, ptr %10, align 4
// ESCAPE-NEXT:   %12 = icmp ne i32 %11, 1405670641
// ESCAPE-NEXT:   br i1 %12, label %_llgo_3, label %_llgo_5
// ESCAPE-EMPTY:
// ESCAPE-NEXT: _llgo_2:                                          ; preds = %_llgo_8
// ESCAPE-NEXT:   %13 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// ESCAPE-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @5, i64 14 }, ptr %13, align 8
// ESCAPE-NEXT:   %14 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %13, 1
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %14)
// ESCAPE-NEXT:   unreachable
// ESCAPE-EMPTY:
// ESCAPE-NEXT: _llgo_3:                                          ; preds = %_llgo_5, %_llgo_1
// ESCAPE-NEXT:   %15 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// ESCAPE-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @6, i64 14 }, ptr %15, align 8
// ESCAPE-NEXT:   %16 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %15, 1
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %16)
// ESCAPE-NEXT:   unreachable
// ESCAPE-EMPTY:
// ESCAPE-NEXT: _llgo_4:                                          ; preds = %_llgo_5
// ESCAPE-NEXT:   ret void
// ESCAPE-EMPTY:
// ESCAPE-NEXT: _llgo_5:                                          ; preds = %_llgo_1
// ESCAPE-NEXT:   %17 = getelementptr inbounds %main.u64parts, ptr %6, i32 0, i32 1
// ESCAPE-NEXT:   %18 = load i32, ptr %17, align 4
// ESCAPE-NEXT:   %19 = icmp ne i32 %18, 1074340347
// ESCAPE-NEXT:   br i1 %19, label %_llgo_3, label %_llgo_4
// ESCAPE-EMPTY:
// ESCAPE-NEXT: _llgo_6:                                          ; preds = %_llgo_0
// ESCAPE-NEXT:   %20 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %1, 1
// ESCAPE-NEXT:   %21 = load double, ptr %20, align 8
// ESCAPE-NEXT:   %22 = insertvalue { double, i1 } undef, double %21, 0
// ESCAPE-NEXT:   %23 = insertvalue { double, i1 } %22, i1 true, 1
// ESCAPE-NEXT:   br label %_llgo_8
// ESCAPE-EMPTY:
// ESCAPE-NEXT: _llgo_7:                                          ; preds = %_llgo_0
// ESCAPE-NEXT:   br label %_llgo_8
// ESCAPE-EMPTY:
// ESCAPE-NEXT: _llgo_8:                                          ; preds = %_llgo_7, %_llgo_6
// ESCAPE-NEXT:   %24 = phi { double, i1 } [ %23, %_llgo_6 ], [ zeroinitializer, %_llgo_7 ]
// ESCAPE-NEXT:   %25 = extractvalue { double, i1 } %24, 0
// ESCAPE-NEXT:   %26 = extractvalue { double, i1 } %24, 1
// ESCAPE-NEXT:   br i1 %26, label %_llgo_1, label %_llgo_2
// ESCAPE-NEXT: }

// CHECK-LABEL: define void @main.check64(%"{{.*}}/runtime/internal/runtime.eface" %0){{.*}} {
// CHECK: icmp eq ptr {{.*}}, @_llgo_float64
// CHECK: load i32, ptr
// CHECK: icmp ne i32 {{.*}}, 1405670641
// CHECK: load i32, ptr
// CHECK: icmp ne i32 {{.*}}, 1074340347
func check64(v any) {
	switch v.(type) {
	case float64:
	default:
		panic("error type f64")
	}
	e := *(*eface)(unsafe.Pointer(&v))
	bits := *(*u64parts)(e.data)
	if bits.lo != pi64lo || bits.hi != pi64hi {
		panic("error bits f64")
	}
}

// CHECK-LABEL: define float @main.f32(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   ret float 0x400921FB60000000
// CHECK-NEXT: }
func f32() float32 {
	return pi
}

// CHECK-LABEL: define double @main.f64(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   ret double 0x400921FB53C8D4F1
// CHECK-NEXT: }
func f64() float64 {
	return pi
}

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call float @main.f32()
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store float %0, ptr %1, align 4
// CHECK-NEXT:   %2 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_float32, ptr undef }, ptr %1, 1
// CHECK-NEXT:   call void @main.check32(%"{{.*}}/runtime/internal/runtime.eface" %2)
// CHECK-NEXT:   %3 = call double @main.f64()
// CHECK-NEXT:   %4 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store double %3, ptr %4, align 8
// CHECK-NEXT:   %5 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_float64, ptr undef }, ptr %4, 1
// CHECK-NEXT:   call void @main.check64(%"{{.*}}/runtime/internal/runtime.eface" %5)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func main() {
	check32(f32())
	check64(f64())
}
