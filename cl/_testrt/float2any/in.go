// LITTEST
package main

import "unsafe"

// CHECK-LINE: @1 = private unnamed_addr constant [14 x i8] c"error type f32", align 1
// CHECK-LINE: @3 = private unnamed_addr constant [14 x i8] c"error bits f32", align 1
// CHECK-LINE: @5 = private unnamed_addr constant [14 x i8] c"error type f64", align 1
// CHECK-LINE: @6 = private unnamed_addr constant [14 x i8] c"error bits f64", align 1

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

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/float2any.check32"(%"{{.*}}/runtime/internal/runtime.eface" %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %0, ptr %1, align 8
// CHECK-NEXT:   %2 = load %"{{.*}}/runtime/internal/runtime.eface", ptr %1, align 8
// CHECK-NEXT:   %3 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %2, 0
// CHECK-NEXT:   %4 = icmp eq ptr %3, @_llgo_float32
// CHECK-NEXT:   br i1 %4, label %_llgo_5, label %_llgo_6
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_7
// CHECK-NEXT:   %5 = alloca %"{{.*}}/cl/_testrt/float2any.eface", align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %5, i8 0, i64 16, i1 false)
// CHECK-NEXT:   %6 = load %"{{.*}}/cl/_testrt/float2any.eface", ptr %1, align 8
// CHECK-NEXT:   store %"{{.*}}/cl/_testrt/float2any.eface" %6, ptr %5, align 8
// CHECK-NEXT:   %7 = icmp eq ptr %5, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %7)
// CHECK-NEXT:   %8 = icmp eq ptr %5, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %8)
// CHECK-NEXT:   %9 = getelementptr inbounds %"{{.*}}/cl/_testrt/float2any.eface", ptr %5, i32 0, i32 1
// CHECK-NEXT:   %10 = load ptr, ptr %9, align 8
// CHECK-NEXT:   %11 = load i32, ptr %10, align 4
// CHECK-NEXT:   %12 = icmp ne i32 %11, 1078530011
// CHECK-NEXT:   br i1 %12, label %_llgo_3, label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_7
// CHECK-NEXT:   %13 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @1, i64 14 }, ptr %13, align 8
// CHECK-NEXT:   %14 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %13, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %14)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_1
// CHECK-NEXT:   %15 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @3, i64 14 }, ptr %15, align 8
// CHECK-NEXT:   %16 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %15, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %16)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_1
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %17 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %2, 1
// CHECK-NEXT:   %18 = load float, ptr %17, align 4
// CHECK-NEXT:   %19 = insertvalue { float, i1 } undef, float %18, 0
// CHECK-NEXT:   %20 = insertvalue { float, i1 } %19, i1 true, 1
// CHECK-NEXT:   br label %_llgo_7
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_0
// CHECK-NEXT:   br label %_llgo_7
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_7:                                          ; preds = %_llgo_6, %_llgo_5
// CHECK-NEXT:   %21 = phi { float, i1 } [ %20, %_llgo_5 ], [ zeroinitializer, %_llgo_6 ]
// CHECK-NEXT:   %22 = extractvalue { float, i1 } %21, 0
// CHECK-NEXT:   %23 = extractvalue { float, i1 } %21, 1
// CHECK-NEXT:   br i1 %23, label %_llgo_1, label %_llgo_2
// CHECK-NEXT: }

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

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/float2any.check64"(%"{{.*}}/runtime/internal/runtime.eface" %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %0, ptr %1, align 8
// CHECK-NEXT:   %2 = load %"{{.*}}/runtime/internal/runtime.eface", ptr %1, align 8
// CHECK-NEXT:   %3 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %2, 0
// CHECK-NEXT:   %4 = icmp eq ptr %3, @_llgo_float64
// CHECK-NEXT:   br i1 %4, label %_llgo_6, label %_llgo_7
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_8
// CHECK-NEXT:   %5 = alloca %"{{.*}}/cl/_testrt/float2any.eface", align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %5, i8 0, i64 16, i1 false)
// CHECK-NEXT:   %6 = load %"{{.*}}/cl/_testrt/float2any.eface", ptr %1, align 8
// CHECK-NEXT:   store %"{{.*}}/cl/_testrt/float2any.eface" %6, ptr %5, align 8
// CHECK-NEXT:   %7 = alloca %"{{.*}}/cl/_testrt/float2any.u64parts", align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %7, i8 0, i64 8, i1 false)
// CHECK-NEXT:   %8 = icmp eq ptr %5, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %8)
// CHECK-NEXT:   %9 = icmp eq ptr %5, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %9)
// CHECK-NEXT:   %10 = getelementptr inbounds %"{{.*}}/cl/_testrt/float2any.eface", ptr %5, i32 0, i32 1
// CHECK-NEXT:   %11 = load ptr, ptr %10, align 8
// CHECK-NEXT:   %12 = load %"{{.*}}/cl/_testrt/float2any.u64parts", ptr %11, align 4
// CHECK-NEXT:   store %"{{.*}}/cl/_testrt/float2any.u64parts" %12, ptr %7, align 4
// CHECK-NEXT:   %13 = icmp eq ptr %7, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %13)
// CHECK-NEXT:   %14 = icmp eq ptr %7, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %14)
// CHECK-NEXT:   %15 = getelementptr inbounds %"{{.*}}/cl/_testrt/float2any.u64parts", ptr %7, i32 0, i32 0
// CHECK-NEXT:   %16 = load i32, ptr %15, align 4
// CHECK-NEXT:   %17 = icmp ne i32 %16, 1405670641
// CHECK-NEXT:   br i1 %17, label %_llgo_3, label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_8
// CHECK-NEXT:   %18 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @5, i64 14 }, ptr %18, align 8
// CHECK-NEXT:   %19 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %18, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %19)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_5, %_llgo_1
// CHECK-NEXT:   %20 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @6, i64 14 }, ptr %20, align 8
// CHECK-NEXT:   %21 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %20, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %21)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_5
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_1
// CHECK-NEXT:   %22 = icmp eq ptr %7, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %22)
// CHECK-NEXT:   %23 = icmp eq ptr %7, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %23)
// CHECK-NEXT:   %24 = getelementptr inbounds %"{{.*}}/cl/_testrt/float2any.u64parts", ptr %7, i32 0, i32 1
// CHECK-NEXT:   %25 = load i32, ptr %24, align 4
// CHECK-NEXT:   %26 = icmp ne i32 %25, 1074340347
// CHECK-NEXT:   br i1 %26, label %_llgo_3, label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %27 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %2, 1
// CHECK-NEXT:   %28 = load double, ptr %27, align 8
// CHECK-NEXT:   %29 = insertvalue { double, i1 } undef, double %28, 0
// CHECK-NEXT:   %30 = insertvalue { double, i1 } %29, i1 true, 1
// CHECK-NEXT:   br label %_llgo_8
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_7:                                          ; preds = %_llgo_0
// CHECK-NEXT:   br label %_llgo_8
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_8:                                          ; preds = %_llgo_7, %_llgo_6
// CHECK-NEXT:   %31 = phi { double, i1 } [ %30, %_llgo_6 ], [ zeroinitializer, %_llgo_7 ]
// CHECK-NEXT:   %32 = extractvalue { double, i1 } %31, 0
// CHECK-NEXT:   %33 = extractvalue { double, i1 } %31, 1
// CHECK-NEXT:   br i1 %33, label %_llgo_1, label %_llgo_2
// CHECK-NEXT: }

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

// CHECK-LABEL: define float @"{{.*}}/cl/_testrt/float2any.f32"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   ret float 0x400921FB60000000
// CHECK-NEXT: }

func f32() float32 {
	return pi
}

// CHECK-LABEL: define double @"{{.*}}/cl/_testrt/float2any.f64"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   ret double 0x400921FB53C8D4F1
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/float2any.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testrt/float2any.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testrt/float2any.init$guard", align 1
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

func f64() float64 {
	return pi
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/float2any.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call float @"{{.*}}/cl/_testrt/float2any.f32"()
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store float %0, ptr %1, align 4
// CHECK-NEXT:   %2 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_float32, ptr undef }, ptr %1, 1
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/float2any.check32"(%"{{.*}}/runtime/internal/runtime.eface" %2)
// CHECK-NEXT:   %3 = call double @"{{.*}}/cl/_testrt/float2any.f64"()
// CHECK-NEXT:   %4 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store double %3, ptr %4, align 8
// CHECK-NEXT:   %5 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_float64, ptr undef }, ptr %4, 1
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/float2any.check64"(%"{{.*}}/runtime/internal/runtime.eface" %5)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

func main() {
	check32(f32())
	check64(f64())
}

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.f32equal"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.f32equal"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.f64equal"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.f64equal"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }
