// LITTEST
package main

/*
#include <stdio.h>
#include <stdlib.h>
#include <math.h>
*/
import "C"
import (
	"fmt"
	"unsafe"
)

// CHECK-LINE: @0 = private unnamed_addr constant [13 x i8] c"Hello, World!", align 1
// CHECK-LINE: @1 = private unnamed_addr constant [29 x i8] c"Converted back to Go string: ", align 1
// CHECK-LINE: @2 = private unnamed_addr constant [23 x i8] c"Length-limited string: ", align 1
// CHECK-LINE: @3 = private unnamed_addr constant [33 x i8] c"Converted back to Go byte slice: ", align 1
// CHECK-LINE: @5 = private unnamed_addr constant [14 x i8] c"sqrt(%v) = %v\0A", align 1
// CHECK-LINE: @6 = private unnamed_addr constant [13 x i8] c"sin(%v) = %v\0A", align 1
// CHECK-LINE: @7 = private unnamed_addr constant [13 x i8] c"cos(%v) = %v\0A", align 1
// CHECK-LINE: @8 = private unnamed_addr constant [13 x i8] c"log(%v) = %v\0A", align 1

func main() {
	// C.CString example
	cstr := C.CString("Hello, World!")
	C.puts(cstr)

	// C.CBytes example
	bytes := []byte{65, 66, 67, 68} // ABCD
	cbytes := C.CBytes(bytes)

	// C.GoString example
	gostr := C.GoString(cstr)
	println("Converted back to Go string: ", gostr)

	// C.GoStringN example (with length limit)
	gostringN := C.GoStringN(cstr, 5) // only take first 5 characters
	println("Length-limited string: ", gostringN)

	// C.GoBytes example
	gobytes := C.GoBytes(cbytes, 4) // 4 is the length
	println("Converted back to Go byte slice: ", gobytes)

	// C math library examples
	x := 2.0
	// Calculate square root
	sqrtResult := C.sqrt(C.double(x))
	fmt.Printf("sqrt(%v) = %v\n", x, float64(sqrtResult))

	// Calculate sine
	sinResult := C.sin(C.double(x))
	fmt.Printf("sin(%v) = %v\n", x, float64(sinResult))

	// Calculate cosine
	cosResult := C.cos(C.double(x))
	fmt.Printf("cos(%v) = %v\n", x, float64(cosResult))

	// Calculate natural logarithm
	logResult := C.log(C.double(x))
	fmt.Printf("log(%v) = %v\n", x, float64(logResult))

	C.free(unsafe.Pointer(cstr))
	C.free(cbytes)
}

// CHECK-LABEL: define double @"{{.*}}/cl/_testgo/cgobasic._Cfunc_cos"(double %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %2 = load ptr, ptr @"{{.*}}/cl/_testgo/cgobasic._cgo_cd50f4724082_Cfunc_cos", align 8
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call double %3(double %0)
// CHECK-NEXT:   ret double %4
// CHECK-NEXT: }

// CHECK-LABEL: define [0 x i8] @"{{.*}}/cl/_testgo/cgobasic._Cfunc_free"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %2 = load ptr, ptr @"{{.*}}/cl/_testgo/cgobasic._cgo_cd50f4724082_Cfunc_free", align 8
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call [0 x i8] %3(ptr %0)
// CHECK-NEXT:   ret [0 x i8] %4
// CHECK-NEXT: }

// CHECK-LABEL: define double @"{{.*}}/cl/_testgo/cgobasic._Cfunc_log"(double %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %2 = load ptr, ptr @"{{.*}}/cl/_testgo/cgobasic._cgo_cd50f4724082_Cfunc_log", align 8
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call double %3(double %0)
// CHECK-NEXT:   ret double %4
// CHECK-NEXT: }

// CHECK-LABEL: define i32 @"{{.*}}/cl/_testgo/cgobasic._Cfunc_puts"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %2 = load ptr, ptr @"{{.*}}/cl/_testgo/cgobasic._cgo_cd50f4724082_Cfunc_puts", align 8
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call i32 %3(ptr %0)
// CHECK-NEXT:   ret i32 %4
// CHECK-NEXT: }

// CHECK-LABEL: define double @"{{.*}}/cl/_testgo/cgobasic._Cfunc_sin"(double %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %2 = load ptr, ptr @"{{.*}}/cl/_testgo/cgobasic._cgo_cd50f4724082_Cfunc_sin", align 8
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call double %3(double %0)
// CHECK-NEXT:   ret double %4
// CHECK-NEXT: }

// CHECK-LABEL: define double @"{{.*}}/cl/_testgo/cgobasic._Cfunc_sqrt"(double %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %2 = load ptr, ptr @"{{.*}}/cl/_testgo/cgobasic._cgo_cd50f4724082_Cfunc_sqrt", align 8
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call double %3(double %0)
// CHECK-NEXT:   ret double %4
// CHECK-NEXT: }

// CHECK-LABEL: define ptr @"{{.*}}/cl/_testgo/cgobasic._Cgo_ptr"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   ret ptr %0
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/cgobasic.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testgo/cgobasic.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testgo/cgobasic.init$guard", align 1
// CHECK-NEXT:   call void @syscall.init()
// CHECK-NEXT:   call void @fmt.init()
// CHECK-NEXT:   store ptr @_cgo_cd50f4724082_Cfunc_cos, ptr @"{{.*}}/cl/_testgo/cgobasic._cgo_cd50f4724082_Cfunc_cos", align 8
// CHECK-NEXT:   store ptr @_cgo_cd50f4724082_Cfunc_free, ptr @"{{.*}}/cl/_testgo/cgobasic._cgo_cd50f4724082_Cfunc_free", align 8
// CHECK-NEXT:   store ptr @_cgo_cd50f4724082_Cfunc_log, ptr @"{{.*}}/cl/_testgo/cgobasic._cgo_cd50f4724082_Cfunc_log", align 8
// CHECK-NEXT:   store ptr @_cgo_cd50f4724082_Cfunc_puts, ptr @"{{.*}}/cl/_testgo/cgobasic._cgo_cd50f4724082_Cfunc_puts", align 8
// CHECK-NEXT:   store ptr @_cgo_cd50f4724082_Cfunc_sin, ptr @"{{.*}}/cl/_testgo/cgobasic._cgo_cd50f4724082_Cfunc_sin", align 8
// CHECK-NEXT:   store ptr @_cgo_cd50f4724082_Cfunc_sqrt, ptr @"{{.*}}/cl/_testgo/cgobasic._cgo_cd50f4724082_Cfunc_sqrt", align 8
// CHECK-NEXT:   store ptr @_cgo_cd50f4724082_Cfunc__Cmalloc, ptr @"{{.*}}/cl/_testgo/cgobasic._cgo_cd50f4724082_Cfunc__Cmalloc", align 8
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/cgobasic.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.CString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 13 })
// CHECK-NEXT:   store ptr %1, ptr %0, align 8
// CHECK-NEXT:   %2 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %3 = call i32 @"{{.*}}/cl/_testgo/cgobasic._Cfunc_puts"(ptr %2)
// CHECK-NEXT:   %4 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 24)
// CHECK-NEXT:   %5 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 4)
// CHECK-NEXT:   %6 = getelementptr inbounds i8, ptr %5, i64 0
// CHECK-NEXT:   store i8 65, ptr %6, align 1
// CHECK-NEXT:   %7 = getelementptr inbounds i8, ptr %5, i64 1
// CHECK-NEXT:   store i8 66, ptr %7, align 1
// CHECK-NEXT:   %8 = getelementptr inbounds i8, ptr %5, i64 2
// CHECK-NEXT:   store i8 67, ptr %8, align 1
// CHECK-NEXT:   %9 = getelementptr inbounds i8, ptr %5, i64 3
// CHECK-NEXT:   store i8 68, ptr %9, align 1
// CHECK-NEXT:   %10 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %5, 0
// CHECK-NEXT:   %11 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %10, i64 4, 1
// CHECK-NEXT:   %12 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %11, i64 4, 2
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" %12, ptr %4, align 8
// CHECK-NEXT:   %13 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %14 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   %15 = getelementptr inbounds { ptr }, ptr %14, i32 0, i32 0
// CHECK-NEXT:   store ptr %4, ptr %15, align 8
// CHECK-NEXT:   %16 = insertvalue { ptr, ptr } { ptr @"{{.*}}/cl/_testgo/cgobasic.main$1", ptr undef }, ptr %14, 1
// CHECK-NEXT:   %17 = extractvalue { ptr, ptr } %16, 1
// CHECK-NEXT:   %18 = extractvalue { ptr, ptr } %16, 0
// CHECK-NEXT:   %19 = call ptr %18(ptr %17)
// CHECK-NEXT:   store ptr %19, ptr %13, align 8
// CHECK-NEXT:   %20 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %21 = call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/runtime/internal/runtime.GoString"(ptr %20)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @1, i64 29 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %21)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %22 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %23 = call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/runtime/internal/runtime.GoStringN"(ptr %22, i64 5)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @2, i64 23 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %23)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %24 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   %25 = getelementptr inbounds { ptr }, ptr %24, i32 0, i32 0
// CHECK-NEXT:   store ptr %13, ptr %25, align 8
// CHECK-NEXT:   %26 = insertvalue { ptr, ptr } { ptr @"{{.*}}/cl/_testgo/cgobasic.main$2", ptr undef }, ptr %24, 1
// CHECK-NEXT:   %27 = extractvalue { ptr, ptr } %26, 1
// CHECK-NEXT:   %28 = extractvalue { ptr, ptr } %26, 0
// CHECK-NEXT:   %29 = call %"{{.*}}/runtime/internal/runtime.Slice" %28(ptr %27)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @3, i64 33 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintSlice"(%"{{.*}}/runtime/internal/runtime.Slice" %29)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %30 = call double @"{{.*}}/cl/_testgo/cgobasic._Cfunc_sqrt"(double 2.000000e+00)
// CHECK-NEXT:   %31 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 32)
// CHECK-NEXT:   %32 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %31, i64 0
// CHECK-NEXT:   %33 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store double 2.000000e+00, ptr %33, align 8
// CHECK-NEXT:   %34 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_float64, ptr undef }, ptr %33, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %34, ptr %32, align 8
// CHECK-NEXT:   %35 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %31, i64 1
// CHECK-NEXT:   %36 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store double %30, ptr %36, align 8
// CHECK-NEXT:   %37 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_float64, ptr undef }, ptr %36, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %37, ptr %35, align 8
// CHECK-NEXT:   %38 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %31, 0
// CHECK-NEXT:   %39 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %38, i64 2, 1
// CHECK-NEXT:   %40 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %39, i64 2, 2
// CHECK-NEXT:   %41 = call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @fmt.Printf(%"{{.*}}/runtime/internal/runtime.String" { ptr @5, i64 14 }, %"{{.*}}/runtime/internal/runtime.Slice" %40)
// CHECK-NEXT:   %42 = call double @"{{.*}}/cl/_testgo/cgobasic._Cfunc_sin"(double 2.000000e+00)
// CHECK-NEXT:   %43 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 32)
// CHECK-NEXT:   %44 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %43, i64 0
// CHECK-NEXT:   %45 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store double 2.000000e+00, ptr %45, align 8
// CHECK-NEXT:   %46 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_float64, ptr undef }, ptr %45, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %46, ptr %44, align 8
// CHECK-NEXT:   %47 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %43, i64 1
// CHECK-NEXT:   %48 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store double %42, ptr %48, align 8
// CHECK-NEXT:   %49 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_float64, ptr undef }, ptr %48, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %49, ptr %47, align 8
// CHECK-NEXT:   %50 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %43, 0
// CHECK-NEXT:   %51 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %50, i64 2, 1
// CHECK-NEXT:   %52 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %51, i64 2, 2
// CHECK-NEXT:   %53 = call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @fmt.Printf(%"{{.*}}/runtime/internal/runtime.String" { ptr @6, i64 13 }, %"{{.*}}/runtime/internal/runtime.Slice" %52)
// CHECK-NEXT:   %54 = call double @"{{.*}}/cl/_testgo/cgobasic._Cfunc_cos"(double 2.000000e+00)
// CHECK-NEXT:   %55 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 32)
// CHECK-NEXT:   %56 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %55, i64 0
// CHECK-NEXT:   %57 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store double 2.000000e+00, ptr %57, align 8
// CHECK-NEXT:   %58 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_float64, ptr undef }, ptr %57, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %58, ptr %56, align 8
// CHECK-NEXT:   %59 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %55, i64 1
// CHECK-NEXT:   %60 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store double %54, ptr %60, align 8
// CHECK-NEXT:   %61 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_float64, ptr undef }, ptr %60, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %61, ptr %59, align 8
// CHECK-NEXT:   %62 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %55, 0
// CHECK-NEXT:   %63 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %62, i64 2, 1
// CHECK-NEXT:   %64 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %63, i64 2, 2
// CHECK-NEXT:   %65 = call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @fmt.Printf(%"{{.*}}/runtime/internal/runtime.String" { ptr @7, i64 13 }, %"{{.*}}/runtime/internal/runtime.Slice" %64)
// CHECK-NEXT:   %66 = call double @"{{.*}}/cl/_testgo/cgobasic._Cfunc_log"(double 2.000000e+00)
// CHECK-NEXT:   %67 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 32)
// CHECK-NEXT:   %68 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %67, i64 0
// CHECK-NEXT:   %69 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store double 2.000000e+00, ptr %69, align 8
// CHECK-NEXT:   %70 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_float64, ptr undef }, ptr %69, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %70, ptr %68, align 8
// CHECK-NEXT:   %71 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %67, i64 1
// CHECK-NEXT:   %72 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store double %66, ptr %72, align 8
// CHECK-NEXT:   %73 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_float64, ptr undef }, ptr %72, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %73, ptr %71, align 8
// CHECK-NEXT:   %74 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %67, 0
// CHECK-NEXT:   %75 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %74, i64 2, 1
// CHECK-NEXT:   %76 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %75, i64 2, 2
// CHECK-NEXT:   %77 = call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @fmt.Printf(%"{{.*}}/runtime/internal/runtime.String" { ptr @8, i64 13 }, %"{{.*}}/runtime/internal/runtime.Slice" %76)
// CHECK-NEXT:   %78 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   %79 = getelementptr inbounds { ptr }, ptr %78, i32 0, i32 0
// CHECK-NEXT:   store ptr %0, ptr %79, align 8
// CHECK-NEXT:   %80 = insertvalue { ptr, ptr } { ptr @"{{.*}}/cl/_testgo/cgobasic.main$3", ptr undef }, ptr %78, 1
// CHECK-NEXT:   %81 = extractvalue { ptr, ptr } %80, 1
// CHECK-NEXT:   %82 = extractvalue { ptr, ptr } %80, 0
// CHECK-NEXT:   call void %82(ptr %81)
// CHECK-NEXT:   %83 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   %84 = getelementptr inbounds { ptr }, ptr %83, i32 0, i32 0
// CHECK-NEXT:   store ptr %13, ptr %84, align 8
// CHECK-NEXT:   %85 = insertvalue { ptr, ptr } { ptr @"{{.*}}/cl/_testgo/cgobasic.main$4", ptr undef }, ptr %83, 1
// CHECK-NEXT:   %86 = extractvalue { ptr, ptr } %85, 1
// CHECK-NEXT:   %87 = extractvalue { ptr, ptr } %85, 0
// CHECK-NEXT:   call void %87(ptr %86)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define ptr @"{{.*}}/cl/_testgo/cgobasic.main$1"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = load { ptr }, ptr %0, align 8
// CHECK-NEXT:   %2 = extractvalue { ptr } %1, 0
// CHECK-NEXT:   %3 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %2, align 8
// CHECK-NEXT:   %4 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" %3, ptr %4, align 8
// CHECK-NEXT:   %5 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"[]_llgo_uint8", ptr undef }, ptr %4, 1
// CHECK-NEXT:   %6 = call ptr @"{{.*}}/runtime/internal/runtime.CBytes"(%"{{.*}}/runtime/internal/runtime.Slice" %3)
// CHECK-NEXT:   ret ptr %6
// CHECK-NEXT: }

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/cl/_testgo/cgobasic.main$2"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = load { ptr }, ptr %0, align 8
// CHECK-NEXT:   %2 = extractvalue { ptr } %1, 0
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_Pointer, ptr undef }, ptr %3, 1
// CHECK-NEXT:   %5 = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/internal/runtime.GoBytes"(ptr %3, i64 4)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.Slice" %5
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/cgobasic.main$3"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = load { ptr }, ptr %0, align 8
// CHECK-NEXT:   %2 = extractvalue { ptr } %1, 0
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_Pointer, ptr undef }, ptr %3, 1
// CHECK-NEXT:   %5 = call [0 x i8] @"{{.*}}/cl/_testgo/cgobasic._Cfunc_free"(ptr %3)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/cgobasic.main$4"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = load { ptr }, ptr %0, align 8
// CHECK-NEXT:   %2 = extractvalue { ptr } %1, 0
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_Pointer, ptr undef }, ptr %3, 1
// CHECK-NEXT:   %5 = call [0 x i8] @"{{.*}}/cl/_testgo/cgobasic._Cfunc_free"(ptr %3)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.f64equal"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.f64equal"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequal8"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.memequal8"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }
