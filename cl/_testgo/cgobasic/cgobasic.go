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

// Check the Go/C conversion helpers and generated wrappers without depending
// on fmt's internal lowering.
// CHECK: [[HELLO_WORLD:@[0-9]+]] = private unnamed_addr constant [13 x i8] c"Hello, World!"

// Every generated wrapper forwards its argument through its initialized cgo
// symbol slot and returns the foreign call result unchanged.
// CHECK-LABEL: define double @main._Cfunc_cos(double %0){{.*}} {
// CHECK: [[COS_SLOT:%.*]] = load ptr, ptr @main._cgo_{{.*}}_Cfunc_cos
// CHECK-NEXT: [[COS_FN:%.*]] = load ptr, ptr [[COS_SLOT]]
// CHECK-NEXT: [[COS_RESULT:%.*]] = call double [[COS_FN]](double %0)
// CHECK-NEXT: ret double [[COS_RESULT]]

// CHECK-LABEL: define [0 x i8] @main._Cfunc_free(ptr %0){{.*}} {
// CHECK: [[FREE_SLOT:%.*]] = load ptr, ptr @main._cgo_{{.*}}_Cfunc_free
// CHECK-NEXT: [[FREE_FN:%.*]] = load ptr, ptr [[FREE_SLOT]]
// CHECK-NEXT: [[FREE_RESULT:%.*]] = call [0 x i8] [[FREE_FN]](ptr %0)
// CHECK-NEXT: ret [0 x i8] [[FREE_RESULT]]

// CHECK-LABEL: define double @main._Cfunc_log(double %0){{.*}} {
// CHECK: [[LOG_SLOT:%.*]] = load ptr, ptr @main._cgo_{{.*}}_Cfunc_log
// CHECK-NEXT: [[LOG_FN:%.*]] = load ptr, ptr [[LOG_SLOT]]
// CHECK-NEXT: [[LOG_RESULT:%.*]] = call double [[LOG_FN]](double %0)
// CHECK-NEXT: ret double [[LOG_RESULT]]

// CHECK-LABEL: define i32 @main._Cfunc_puts(ptr %0){{.*}} {
// CHECK: [[PUTS_SLOT:%.*]] = load ptr, ptr @main._cgo_{{.*}}_Cfunc_puts
// CHECK-NEXT: [[PUTS_FN:%.*]] = load ptr, ptr [[PUTS_SLOT]]
// CHECK-NEXT: [[PUTS_RESULT:%.*]] = call i32 [[PUTS_FN]](ptr %0)
// CHECK-NEXT: ret i32 [[PUTS_RESULT]]

// CHECK-LABEL: define double @main._Cfunc_sin(double %0){{.*}} {
// CHECK: [[SIN_SLOT:%.*]] = load ptr, ptr @main._cgo_{{.*}}_Cfunc_sin
// CHECK-NEXT: [[SIN_FN:%.*]] = load ptr, ptr [[SIN_SLOT]]
// CHECK-NEXT: [[SIN_RESULT:%.*]] = call double [[SIN_FN]](double %0)
// CHECK-NEXT: ret double [[SIN_RESULT]]

// CHECK-LABEL: define double @main._Cfunc_sqrt(double %0){{.*}} {
// CHECK: [[SQRT_SLOT:%.*]] = load ptr, ptr @main._cgo_{{.*}}_Cfunc_sqrt
// CHECK-NEXT: [[SQRT_FN:%.*]] = load ptr, ptr [[SQRT_SLOT]]
// CHECK-NEXT: [[SQRT_RESULT:%.*]] = call double [[SQRT_FN]](double %0)
// CHECK-NEXT: ret double [[SQRT_RESULT]]

// CHECK-LABEL: define void @main.init(){{.*}} {
// CHECK: store ptr @_cgo_{{.*}}_Cfunc_cos, ptr @main._cgo_{{.*}}_Cfunc_cos
// CHECK: store ptr @_cgo_{{.*}}_Cfunc_free, ptr @main._cgo_{{.*}}_Cfunc_free
// CHECK: store ptr @_cgo_{{.*}}_Cfunc_log, ptr @main._cgo_{{.*}}_Cfunc_log
// CHECK: store ptr @_cgo_{{.*}}_Cfunc_puts, ptr @main._cgo_{{.*}}_Cfunc_puts
// CHECK: store ptr @_cgo_{{.*}}_Cfunc_sin, ptr @main._cgo_{{.*}}_Cfunc_sin
// CHECK: store ptr @_cgo_{{.*}}_Cfunc_sqrt, ptr @main._cgo_{{.*}}_Cfunc_sqrt

// CHECK-LABEL: define void @main.main(){{.*}} {
// CString is stored once, reused by puts/GoString/GoStringN, and later freed.
// CHECK: [[CSTR_SLOT:%.*]] = call ptr @"{{.*}}AllocZ"(i64 8)
// CHECK-NEXT: [[CSTR:%.*]] = call ptr @"{{.*}}CString"(%"{{.*}}String" { ptr [[HELLO_WORLD]], i64 13 })
// CHECK-NEXT: store ptr [[CSTR]], ptr [[CSTR_SLOT]]
// CHECK-NEXT: [[CSTR_FOR_PUTS:%.*]] = load ptr, ptr [[CSTR_SLOT]]
// CHECK-NEXT: call i32 @main._Cfunc_puts(ptr [[CSTR_FOR_PUTS]])
// The ABCD []byte is passed through the captured CBytes helper and stored.
// CHECK: [[BYTES_SLOT:%.*]] = call ptr @"{{.*}}AllocZ"(i64 24)
// CHECK: [[CBYTES_SLOT:%.*]] = call ptr @"{{.*}}AllocZ"(i64 8)
// CHECK: [[CBYTES_ENV:%.*]] = call ptr @"{{.*}}AllocU"(i64 8)
// CHECK: store ptr [[BYTES_SLOT]], ptr {{%.*}}
// CHECK: [[CBYTES_CLOSURE:%.*]] = insertvalue { ptr, ptr } { ptr @"main.main$1", ptr undef }, ptr [[CBYTES_ENV]], 1
// CHECK: [[CBYTES_CALL_ENV:%.*]] = extractvalue { ptr, ptr } [[CBYTES_CLOSURE]], 1
// CHECK: [[CBYTES_CALL_FN:%.*]] = extractvalue { ptr, ptr } [[CBYTES_CLOSURE]], 0
// CHECK: [[CBYTES_CODE:%.*]] = call ptr asm "", "=r,0"(ptr [[CBYTES_CALL_FN]])
// CHECK-NEXT: [[CBYTES:%.*]] = call ptr [[CBYTES_CODE]](ptr {{(nest|swiftself)}} [[CBYTES_CALL_ENV]])
// CHECK-NEXT: store ptr [[CBYTES]], ptr [[CBYTES_SLOT]]
// CHECK: [[CSTR_FOR_GO:%.*]] = load ptr, ptr [[CSTR_SLOT]]
// CHECK-NEXT: [[GO_STRING:%.*]] = call %"{{.*}}String" @"{{.*}}GoString"(ptr [[CSTR_FOR_GO]])
// CHECK: [[CSTR_FOR_GON:%.*]] = load ptr, ptr [[CSTR_SLOT]]
// CHECK-NEXT: [[GO_STRING_N:%.*]] = call %"{{.*}}String" @"{{.*}}GoStringN"(ptr [[CSTR_FOR_GON]], i64 5)
// CHECK: [[GOBYTES_ENV:%.*]] = call ptr @"{{.*}}AllocU"(i64 8)
// CHECK: store ptr [[CBYTES_SLOT]], ptr {{%.*}}
// CHECK: [[GOBYTES_CLOSURE:%.*]] = insertvalue { ptr, ptr } { ptr @"main.main$2", ptr undef }, ptr [[GOBYTES_ENV]], 1
// CHECK: [[GOBYTES_CALL_ENV:%.*]] = extractvalue { ptr, ptr } [[GOBYTES_CLOSURE]], 1
// CHECK: [[GOBYTES_CALL_FN:%.*]] = extractvalue { ptr, ptr } [[GOBYTES_CLOSURE]], 0
// CHECK: [[GOBYTES_CODE:%.*]] = call ptr asm "", "=r,0"(ptr [[GOBYTES_CALL_FN]])
// CHECK-NEXT: [[GO_BYTES:%.*]] = call %"{{.*}}Slice" [[GOBYTES_CODE]](ptr {{(nest|swiftself)}} [[GOBYTES_CALL_ENV]])
// One libm result is followed through boxing into fmt.Printf; the other
// wrappers already prove their own argument/result forwarding above.
// CHECK: [[SQRT_CALL:%.*]] = call double @main._Cfunc_sqrt(double 2.000000e+00)
// CHECK: [[SQRT_ARGS:%.*]] = call ptr @"{{.*}}AllocZ"(i64 32)
// CHECK: [[SQRT_X_BOX:%.*]] = call ptr @"{{.*}}AllocU"(i64 8)
// CHECK: store double 2.000000e+00, ptr [[SQRT_X_BOX]]
// CHECK: [[SQRT_RESULT_SLOT:%.*]] = getelementptr inbounds %"{{.*}}eface", ptr [[SQRT_ARGS]], i64 1
// CHECK-NEXT: [[SQRT_RESULT_BOX:%.*]] = call ptr @"{{.*}}AllocU"(i64 8)
// CHECK-NEXT: store double [[SQRT_CALL]], ptr [[SQRT_RESULT_BOX]]
// CHECK: call { i64, %"{{.*}}iface" } @fmt.Printf
// CHECK: call double @main._Cfunc_sin(double 2.000000e+00)
// CHECK: call double @main._Cfunc_cos(double 2.000000e+00)
// CHECK: call double @main._Cfunc_log(double 2.000000e+00)

// CHECK-LABEL: define ptr @"main.main$1"(ptr {{(nest|swiftself)}} %0){{.*}} {
// CHECK: [[CB_CAPTURE:%.*]] = load { ptr }, ptr %0
// CHECK-NEXT: [[CB_SLOT:%.*]] = extractvalue { ptr } [[CB_CAPTURE]], 0
// CHECK-NEXT: [[CB_SLICE:%.*]] = load %"{{.*}}Slice", ptr [[CB_SLOT]]
// CHECK: [[CB_RESULT:%.*]] = call ptr @"{{.*}}CBytes"(%"{{.*}}Slice" [[CB_SLICE]])
// CHECK-NEXT: ret ptr [[CB_RESULT]]

// CHECK-LABEL: define %"{{.*}}Slice" @"main.main$2"(ptr {{(nest|swiftself)}} %0){{.*}} {
// CHECK: [[GB_CAPTURE:%.*]] = load { ptr }, ptr %0
// CHECK-NEXT: [[GB_SLOT:%.*]] = extractvalue { ptr } [[GB_CAPTURE]], 0
// CHECK-NEXT: [[GB_PTR:%.*]] = load ptr, ptr [[GB_SLOT]]
// CHECK: [[GB_RESULT:%.*]] = call %"{{.*}}Slice" @"{{.*}}GoBytes"(ptr [[GB_PTR]], i64 4)
// CHECK-NEXT: ret %"{{.*}}Slice" [[GB_RESULT]]

// CHECK-LABEL: define void @"main.main$3"(ptr {{(nest|swiftself)}} %0){{.*}} {
// CHECK: [[FC_CAPTURE:%.*]] = load { ptr }, ptr %0
// CHECK-NEXT: [[FC_SLOT:%.*]] = extractvalue { ptr } [[FC_CAPTURE]], 0
// CHECK-NEXT: [[FC_PTR:%.*]] = load ptr, ptr [[FC_SLOT]]
// CHECK: call [0 x i8] @main._Cfunc_free(ptr [[FC_PTR]])

// CHECK-LABEL: define void @"main.main$4"(ptr {{(nest|swiftself)}} %0){{.*}} {
// CHECK: [[FB_CAPTURE:%.*]] = load { ptr }, ptr %0
// CHECK-NEXT: [[FB_SLOT:%.*]] = extractvalue { ptr } [[FB_CAPTURE]], 0
// CHECK-NEXT: [[FB_PTR:%.*]] = load ptr, ptr [[FB_SLOT]]
// CHECK: call [0 x i8] @main._Cfunc_free(ptr [[FB_PTR]])

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
