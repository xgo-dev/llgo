// LITTEST
package main

import (
	"unsafe"

	"github.com/goplus/lib/c"
	"github.com/goplus/lib/py"
	"github.com/goplus/lib/py/math"
	"github.com/goplus/lib/py/std"
)

// CHECK-LINE: @0 = private unnamed_addr constant [5 x i8] c"world", align 1
// CHECK-LINE: @1 = private unnamed_addr constant [5 x i8] c"hello", align 1
// CHECK-LINE: @2 = private unnamed_addr constant [3 x i8] c"pi\00", align 1
// CHECK-LINE: @3 = private unnamed_addr constant [14 x i8] c"lens = %d %d\0A\00", align 1
// CHECK-LINE: @4 = private unnamed_addr constant [14 x i8] c"ptrs = %d %d\0A\00", align 1
// CHECK-LINE: @5 = private unnamed_addr constant [12 x i8] c"pi = %.15g\0A\00", align 1
// CHECK-LINE: @6 = private unnamed_addr constant [3 x i8] c"pi\00", align 1
// CHECK-LINE: @7 = private unnamed_addr constant [4 x i8] c"abs\00", align 1
// CHECK-LINE: @8 = private unnamed_addr constant [6 x i8] c"print\00", align 1

func main() {
	v := 100
	x := py.List(true, false, 1, float32(2.1), 3.1, uint(4), 1+2i, complex64(3+4i),
		"hello", []byte("world"), [...]byte{1, 2, 3}, [...]byte{}, &v, unsafe.Pointer(&v))
	y := py.List(std.Abs, std.Print, math.Pi)
	ptr := uintptr(unsafe.Pointer(&v))
	c.Printf(c.Str("lens = %d %d\n"), x.ListLen(), y.ListLen())
	c.Printf(c.Str("ptrs = %d %d\n"), intOf(x.ListItem(12).Uintptr() == ptr), intOf(x.ListItem(13).Uintptr() == ptr))
	c.Printf(c.Str("pi = %.15g\n"), math.Pi.Float64())
}

func intOf(ok bool) int {
	if ok {
		return 1
	}
	return 0
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testpy/list.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testpy/list.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testpy/list.init$guard", align 1
// CHECK-NEXT:   call void @"github.com/goplus/lib/py/math.init"()
// CHECK-NEXT:   call void @"github.com/goplus/lib/py/std.init"()
// CHECK-NEXT:   %1 = load ptr, ptr @__llgo_py.builtins, align 8
// CHECK-NEXT:   call void (ptr, ...) @llgoLoadPyModSyms(ptr %1, ptr @7, ptr @__llgo_py.builtins.abs, ptr @8, ptr @__llgo_py.builtins.print, ptr null)
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testpy/list.intOf"(i1 %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   br i1 %0, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   ret i64 1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   ret i64 0
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testpy/list.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   store i64 100, ptr %0, align 8
// CHECK-NEXT:   %1 = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/internal/runtime.StringToBytes"(%"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 5 })
// CHECK-NEXT:   %2 = alloca [3 x i8], align 1
// CHECK-NEXT:   call void @llvm.memset(ptr %2, i8 0, i64 3, i1 false)
// CHECK-NEXT:   %3 = getelementptr inbounds i8, ptr %2, i64 0
// CHECK-NEXT:   %4 = getelementptr inbounds i8, ptr %2, i64 1
// CHECK-NEXT:   %5 = getelementptr inbounds i8, ptr %2, i64 2
// CHECK-NEXT:   store i8 1, ptr %3, align 1
// CHECK-NEXT:   store i8 2, ptr %4, align 1
// CHECK-NEXT:   store i8 3, ptr %5, align 1
// CHECK-NEXT:   %6 = load [3 x i8], ptr %2, align 1
// CHECK-NEXT:   %7 = call ptr @PyList_New(i64 14)
// CHECK-NEXT:   %8 = call ptr @PyBool_FromLong(i32 -1)
// CHECK-NEXT:   %9 = call i32 @PyList_SetItem(ptr %7, i64 0, ptr %8)
// CHECK-NEXT:   %10 = call ptr @PyBool_FromLong(i32 0)
// CHECK-NEXT:   %11 = call i32 @PyList_SetItem(ptr %7, i64 1, ptr %10)
// CHECK-NEXT:   %12 = call ptr @PyLong_FromLongLong(i64 1)
// CHECK-NEXT:   %13 = call i32 @PyList_SetItem(ptr %7, i64 2, ptr %12)
// CHECK-NEXT:   %14 = call ptr @PyFloat_FromDouble(double 0x4000CCCCC0000000)
// CHECK-NEXT:   %15 = call i32 @PyList_SetItem(ptr %7, i64 3, ptr %14)
// CHECK-NEXT:   %16 = call ptr @PyFloat_FromDouble(double 3.100000e+00)
// CHECK-NEXT:   %17 = call i32 @PyList_SetItem(ptr %7, i64 4, ptr %16)
// CHECK-NEXT:   %18 = call ptr @PyLong_FromUnsignedLongLong(i64 4)
// CHECK-NEXT:   %19 = call i32 @PyList_SetItem(ptr %7, i64 5, ptr %18)
// CHECK-NEXT:   %20 = call ptr @PyComplex_FromDoubles(double 1.000000e+00, double 2.000000e+00)
// CHECK-NEXT:   %21 = call i32 @PyList_SetItem(ptr %7, i64 6, ptr %20)
// CHECK-NEXT:   %22 = call ptr @PyComplex_FromDoubles(double 3.000000e+00, double 4.000000e+00)
// CHECK-NEXT:   %23 = call i32 @PyList_SetItem(ptr %7, i64 7, ptr %22)
// CHECK-NEXT:   %24 = call ptr @PyUnicode_FromStringAndSize(ptr @1, i64 5)
// CHECK-NEXT:   %25 = call i32 @PyList_SetItem(ptr %7, i64 8, ptr %24)
// CHECK-NEXT:   %26 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %1, 0
// CHECK-NEXT:   %27 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %1, 1
// CHECK-NEXT:   %28 = call ptr @PyByteArray_FromStringAndSize(ptr %26, i64 %27)
// CHECK-NEXT:   %29 = call i32 @PyList_SetItem(ptr %7, i64 9, ptr %28)
// CHECK-NEXT:   %30 = alloca [3 x i8], align 1
// CHECK-NEXT:   call void @llvm.memset(ptr %30, i8 0, i64 3, i1 false)
// CHECK-NEXT:   store [3 x i8] %6, ptr %30, align 1
// CHECK-NEXT:   %31 = getelementptr inbounds ptr, ptr %30, i64 0
// CHECK-NEXT:   %32 = call ptr @PyBytes_FromStringAndSize(ptr %31, i64 3)
// CHECK-NEXT:   %33 = call i32 @PyList_SetItem(ptr %7, i64 10, ptr %32)
// CHECK-NEXT:   %34 = call ptr @PyBytes_FromStringAndSize(ptr null, i64 0)
// CHECK-NEXT:   %35 = call i32 @PyList_SetItem(ptr %7, i64 11, ptr %34)
// CHECK-NEXT:   %36 = ptrtoint ptr %0 to i64
// CHECK-NEXT:   %37 = call ptr @PyLong_FromUnsignedLongLong(i64 %36)
// CHECK-NEXT:   %38 = call i32 @PyList_SetItem(ptr %7, i64 12, ptr %37)
// CHECK-NEXT:   %39 = ptrtoint ptr %0 to i64
// CHECK-NEXT:   %40 = call ptr @PyLong_FromUnsignedLongLong(i64 %39)
// CHECK-NEXT:   %41 = call i32 @PyList_SetItem(ptr %7, i64 13, ptr %40)
// CHECK-NEXT:   %42 = load ptr, ptr @__llgo_py.math, align 8
// CHECK-NEXT:   %43 = call ptr @PyObject_GetAttrString(ptr %42, ptr @2)
// CHECK-NEXT:   %44 = call ptr @PyList_New(i64 3)
// CHECK-NEXT:   %45 = load ptr, ptr @__llgo_py.builtins.abs, align 8
// CHECK-NEXT:   %46 = call i32 @PyList_SetItem(ptr %44, i64 0, ptr %45)
// CHECK-NEXT:   %47 = load ptr, ptr @__llgo_py.builtins.print, align 8
// CHECK-NEXT:   %48 = call i32 @PyList_SetItem(ptr %44, i64 1, ptr %47)
// CHECK-NEXT:   %49 = call i32 @PyList_SetItem(ptr %44, i64 2, ptr %43)
// CHECK-NEXT:   %50 = ptrtoint ptr %0 to i64
// CHECK-NEXT:   %51 = call i64 @PyList_Size(ptr %7)
// CHECK-NEXT:   %52 = call i64 @PyList_Size(ptr %44)
// CHECK-NEXT:   %53 = call i32 (ptr, ...) @printf(ptr @3, i64 %51, i64 %52)
// CHECK-NEXT:   %54 = call ptr @PyList_GetItem(ptr %7, i64 12)
// CHECK-NEXT:   %55 = call i64 @PyLong_AsSize_t(ptr %54)
// CHECK-NEXT:   %56 = icmp eq i64 %55, %50
// CHECK-NEXT:   %57 = call i64 @"{{.*}}/cl/_testpy/list.intOf"(i1 %56)
// CHECK-NEXT:   %58 = call ptr @PyList_GetItem(ptr %7, i64 13)
// CHECK-NEXT:   %59 = call i64 @PyLong_AsSize_t(ptr %58)
// CHECK-NEXT:   %60 = icmp eq i64 %59, %50
// CHECK-NEXT:   %61 = call i64 @"{{.*}}/cl/_testpy/list.intOf"(i1 %60)
// CHECK-NEXT:   %62 = call i32 (ptr, ...) @printf(ptr @4, i64 %57, i64 %61)
// CHECK-NEXT:   %63 = load ptr, ptr @__llgo_py.math, align 8
// CHECK-NEXT:   %64 = call ptr @PyObject_GetAttrString(ptr %63, ptr @6)
// CHECK-NEXT:   %65 = call double @PyFloat_AsDouble(ptr %64)
// CHECK-NEXT:   %66 = call i32 (ptr, ...) @printf(ptr @5, double %65)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
