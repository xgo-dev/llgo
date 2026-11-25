// LITTEST
// Scope: common
package main

/*
#include <stdlib.h>

static int add(int a, int b) { return a + b; }
static int macro_object = 7;
#define TEST_OBJECT (&macro_object)
static int read_object(const int *p) { return *p; }

typedef struct { int value; } wrapped_object;
static wrapped_object *new_wrapped(int value) {
	wrapped_object *object = (wrapped_object *)malloc(sizeof(wrapped_object));
	object->value = value;
	return object;
}
static int read_wrapped(const wrapped_object *object) { return object->value; }
*/
import "C"

import "unsafe"

// One primitive wrapper is sufficient to prove cgo's symbol-slot forwarding;
// aggregate wrappers and callbacks have their own bounded owners.
// CHECK-LABEL: define i32 @main._Cfunc_add(i32 %0, i32 %1){{.*}} {
// CHECK: [[ADD_SLOT:%[0-9]+]] = load ptr, ptr @main._cgo_{{.*}}_Cfunc_add
// CHECK-NEXT: [[ADD_FN:%[0-9]+]] = load ptr, ptr [[ADD_SLOT]]
// CHECK-NEXT: [[ADD_RESULT:%[0-9]+]] = call i32 [[ADD_FN]](i32 %0, i32 %1)
// CHECK-NEXT: ret i32 [[ADD_RESULT]]

// Keep each compiler-owned conversion helper once, and carry the converted
// values into ordinary Go validation. TEST_OBJECT exercises an object-like C
// macro getter without pulling Python into the compiler fixture.
// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: [[CSTRING:%[0-9]+]] = call ptr @"{{.*}}CString"(%"{{.*}}String" {{.*}})
// CHECK-NEXT: store ptr [[CSTRING]], ptr [[CSTRING_SLOT:%[0-9]+]]
// CHECK: [[CSTRING_FOR_GO:%[0-9]+]] = load ptr, ptr [[CSTRING_SLOT]]
// CHECK-NEXT: [[GOSTRING:%[0-9]+]] = call %"{{.*}}String" @"{{.*}}GoString"(ptr [[CSTRING_FOR_GO]])
// CHECK: [[CSTRING_FOR_N:%[0-9]+]] = load ptr, ptr [[CSTRING_SLOT]]
// CHECK-NEXT: call %"{{.*}}String" @"{{.*}}GoStringN"(ptr [[CSTRING_FOR_N]], i64 2)
// CHECK: [[CBYTES:%[0-9]+]] = call ptr @"main.main$1"()
// CHECK: call %"{{.*}}Slice" %__llgo_funcval_code
// CHECK: [[SUM:%[0-9]+]] = call i32 @main._Cfunc_add(i32 20, i32 22)
// CHECK: [[MACRO:%[0-9]+]] = call ptr @main._Cmacro_TEST_OBJECT()
// CHECK: [[OBJECT:%[0-9]+]] = call i32 @main._Cfunc_read_object(ptr [[MACRO]])
// A returned C pointer converted through unsafe.Pointer and immediately passed
// back to C makes cgo generate _Cgo_use. LLGo must recognize and consume that
// marker as an intrinsic, leaving no unresolved call in package IR.
// CHECK: [[WRAPPED:%[0-9]+]] = call ptr @main._Cfunc_new_wrapped(i32 11)
// CHECK-NOT: _Cgo_use
// CHECK: call i32 @main._Cfunc_read_wrapped(ptr [[WRAPPED]])

// CBytes and GoBytes are emitted in compiler-generated expression helpers.
// CHECK-LABEL: define ptr @"main.main$1"(){{.*}} {
// CHECK: [[BYTE_LEN:%[0-9]+]] = insertvalue %"{{.*}}Slice" {{.*}}, i64 3, 1
// CHECK-NEXT: [[BYTE_SLICE:%[0-9]+]] = insertvalue %"{{.*}}Slice" [[BYTE_LEN]], i64 3, 2
// CHECK: [[CBYTES_RESULT:%[0-9]+]] = call ptr @"{{.*}}CBytes"(%"{{.*}}Slice" [[BYTE_SLICE]])
// CHECK-NEXT: ret ptr [[CBYTES_RESULT]]

// CHECK-LABEL: define %"{{.*}}Slice" @"main.main$2"({{.*}}){{.*}} {
// CHECK: [[CBYTES_ARG:%[0-9]+]] = load ptr, ptr %{{[0-9]+}}
// CHECK: [[GOBYTES_RESULT:%[0-9]+]] = call %"{{.*}}Slice" @"{{.*}}GoBytes"(ptr [[CBYTES_ARG]], i64 3)
// CHECK-NEXT: ret %"{{.*}}Slice" [[GOBYTES_RESULT]]
func main() {
	text := C.CString("llgo")
	if C.GoString(text) != "llgo" {
		panic("CString/GoString")
	}
	if C.GoStringN(text, 2) != "ll" {
		panic("GoStringN")
	}

	bytes := C.CBytes([]byte{1, 2, 3})
	roundTrip := C.GoBytes(bytes, 3)
	if len(roundTrip) != 3 || roundTrip[0] != 1 || roundTrip[2] != 3 {
		panic("CBytes/GoBytes")
	}

	if C.add(20, 22) != 42 {
		panic("cgo wrapper")
	}
	if C.read_object(C.TEST_OBJECT) != 7 {
		panic("object-like macro")
	}
	if C.read_wrapped((*C.wrapped_object)(unsafe.Pointer(C.new_wrapped(11)))) != 11 {
		panic("C pointer conversion")
	}

	// The conversion helpers allocate C storage. Cleanup mechanics are checked
	// by cgodefer; direct calls here avoid another defer CFG.
	C.free(unsafe.Pointer(text))
	C.free(bytes)
}
