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

// CHECK-LABEL: define double @"{{.*}}/cl/_testgo/cgobasic._Cfunc_cos"(double %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %2 = load ptr, ptr @"{{.*}}/cl/_testgo/cgobasic._cgo_{{.*}}_Cfunc_cos", align 8
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call double %3(double %0)
// CHECK-NEXT:   ret double %4
// CHECK-NEXT: }

// CHECK-LABEL: define [0 x i8] @"{{.*}}/cl/_testgo/cgobasic._Cfunc_free"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %2 = load ptr, ptr @"{{.*}}/cl/_testgo/cgobasic._cgo_{{.*}}_Cfunc_free", align 8
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call [0 x i8] %3(ptr %0)
// CHECK-NEXT:   ret [0 x i8] %4
// CHECK-NEXT: }

// CHECK-LABEL: define double @"{{.*}}/cl/_testgo/cgobasic._Cfunc_log"(double %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %2 = load ptr, ptr @"{{.*}}/cl/_testgo/cgobasic._cgo_{{.*}}_Cfunc_log", align 8
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call double %3(double %0)
// CHECK-NEXT:   ret double %4
// CHECK-NEXT: }

// CHECK-LABEL: define i32 @"{{.*}}/cl/_testgo/cgobasic._Cfunc_puts"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %2 = load ptr, ptr @"{{.*}}/cl/_testgo/cgobasic._cgo_{{.*}}_Cfunc_puts", align 8
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call i32 %3(ptr %0)
// CHECK-NEXT:   ret i32 %4
// CHECK-NEXT: }

// CHECK-LABEL: define double @"{{.*}}/cl/_testgo/cgobasic._Cfunc_sin"(double %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %2 = load ptr, ptr @"{{.*}}/cl/_testgo/cgobasic._cgo_{{.*}}_Cfunc_sin", align 8
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call double %3(double %0)
// CHECK-NEXT:   ret double %4
// CHECK-NEXT: }

// CHECK-LABEL: define double @"{{.*}}/cl/_testgo/cgobasic._Cfunc_sqrt"(double %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %2 = load ptr, ptr @"{{.*}}/cl/_testgo/cgobasic._cgo_{{.*}}_Cfunc_sqrt", align 8
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call double %3(double %0)
// CHECK-NEXT:   ret double %4
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
// CHECK-NEXT:   store ptr @_cgo_{{.*}}_Cfunc_cos, ptr @"{{.*}}/cl/_testgo/cgobasic._cgo_{{.*}}_Cfunc_cos", align 8
// CHECK-NEXT:   store ptr @_cgo_{{.*}}_Cfunc_free, ptr @"{{.*}}/cl/_testgo/cgobasic._cgo_{{.*}}_Cfunc_free", align 8
// CHECK-NEXT:   store ptr @_cgo_{{.*}}_Cfunc_log, ptr @"{{.*}}/cl/_testgo/cgobasic._cgo_{{.*}}_Cfunc_log", align 8
// CHECK-NEXT:   store ptr @_cgo_{{.*}}_Cfunc_puts, ptr @"{{.*}}/cl/_testgo/cgobasic._cgo_{{.*}}_Cfunc_puts", align 8
// CHECK-NEXT:   store ptr @_cgo_{{.*}}_Cfunc_sin, ptr @"{{.*}}/cl/_testgo/cgobasic._cgo_{{.*}}_Cfunc_sin", align 8
// CHECK-NEXT:   store ptr @_cgo_{{.*}}_Cfunc_sqrt, ptr @"{{.*}}/cl/_testgo/cgobasic._cgo_{{.*}}_Cfunc_sqrt", align 8
// CHECK-NEXT:   store ptr @_cgo_{{.*}}_Cfunc__Cmalloc, ptr @"{{.*}}/cl/_testgo/cgobasic._cgo_{{.*}}_Cfunc__Cmalloc", align 8
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/cgobasic.main"(){{.*}} {
// CHECK:   %{{.*}} = call i32 @"{{.*}}/cl/_testgo/cgobasic._Cfunc_puts"(ptr %{{.*}})
// CHECK:   %{{.*}} = call double @"{{.*}}/cl/_testgo/cgobasic._Cfunc_sqrt"(double 2.000000e+00)
// CHECK:   %{{.*}} = call double @"{{.*}}/cl/_testgo/cgobasic._Cfunc_sin"(double 2.000000e+00)
// CHECK:   %{{.*}} = call double @"{{.*}}/cl/_testgo/cgobasic._Cfunc_cos"(double 2.000000e+00)
// CHECK:   %{{.*}} = call double @"{{.*}}/cl/_testgo/cgobasic._Cfunc_log"(double 2.000000e+00)
// CHECK:   %{{.*}} = insertvalue { ptr, ptr } { ptr @"{{.*}}/cl/_testgo/cgobasic.main$3", ptr undef }, ptr %{{.*}}, 1
// CHECK:   %{{.*}} = insertvalue { ptr, ptr } { ptr @"{{.*}}/cl/_testgo/cgobasic.main$4", ptr undef }, ptr %{{.*}}, 1
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
