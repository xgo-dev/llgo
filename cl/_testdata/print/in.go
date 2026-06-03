// LITTEST
package main

import (
	"unsafe"

	"github.com/goplus/lib/c"
)

// CHECK-LINE: @0 = private unnamed_addr constant [3 x i8] c"%c\00", align 1
// CHECK-LINE: @1 = private unnamed_addr constant [4 x i8] c"llgo", align 1
// CHECK-LINE: @4 = private unnamed_addr constant [10 x i8] c"check bool", align 1
// CHECK-LINE: @7 = private unnamed_addr constant [8 x i8] c"check &^", align 1
// CHECK-LINE: @21 = private unnamed_addr constant [1 x i8] c"(", align 1
// CHECK-LINE: @22 = private unnamed_addr constant [2 x i8] c"i)", align 1
// CHECK-LINE: @23 = private unnamed_addr constant [4 x i8] c"true", align 1
// CHECK-LINE: @24 = private unnamed_addr constant [5 x i8] c"false", align 1
// CHECK-LINE: @25 = private unnamed_addr constant [3 x i8] c"NaN", align 1
// CHECK-LINE: @26 = private unnamed_addr constant [4 x i8] c"+Inf", align 1
// CHECK-LINE: @27 = private unnamed_addr constant [4 x i8] c"-Inf", align 1
// CHECK-LINE: @28 = private unnamed_addr constant [16 x i8] c"0123456789abcdef", align 1
// CHECK-LINE: @29 = private unnamed_addr constant [1 x i8] c"-", align 1
// CHECK-LINE: @30 = private unnamed_addr constant [1 x i8] c" ", align 1
// CHECK-LINE: @31 = private unnamed_addr constant [1 x i8] c"\0A", align 1

var minhexdigits = 0

type slice struct {
	array unsafe.Pointer
	len   int
	cap   int
}

type stringStruct struct {
	str unsafe.Pointer
	len int
}

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/cl/_testdata/print.bytes"(%"{{.*}}/runtime/internal/runtime.String" %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" %0, ptr %1, align 8
// CHECK-NEXT:   %2 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 24)
// CHECK-NEXT:   %3 = call ptr @"{{.*}}/cl/_testdata/print.stringStructOf"(ptr %1)
// CHECK-NEXT:   %4 = getelementptr inbounds %"{{.*}}/cl/_testdata/print.stringStruct", ptr %3, i32 0, i32 0
// CHECK-NEXT:   %5 = load ptr, ptr %4, align 8
// CHECK-NEXT:   %6 = getelementptr inbounds %"{{.*}}/cl/_testdata/print.slice", ptr %2, i32 0, i32 0
// CHECK-NEXT:   store ptr %5, ptr %6, align 8
// CHECK-NEXT:   %7 = getelementptr inbounds %"{{.*}}/cl/_testdata/print.stringStruct", ptr %3, i32 0, i32 1
// CHECK-NEXT:   %8 = load i64, ptr %7, align 8
// CHECK-NEXT:   %9 = getelementptr inbounds %"{{.*}}/cl/_testdata/print.slice", ptr %2, i32 0, i32 1
// CHECK-NEXT:   store i64 %8, ptr %9, align 8
// CHECK-NEXT:   %10 = getelementptr inbounds %"{{.*}}/cl/_testdata/print.stringStruct", ptr %3, i32 0, i32 1
// CHECK-NEXT:   %11 = load i64, ptr %10, align 8
// CHECK-NEXT:   %12 = getelementptr inbounds %"{{.*}}/cl/_testdata/print.slice", ptr %2, i32 0, i32 2
// CHECK-NEXT:   store i64 %11, ptr %12, align 8
// CHECK-NEXT:   %13 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %2, align 8
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.Slice" %13
// CHECK-NEXT: }
func bytes(s string) (ret []byte) {
	rp := (*slice)(unsafe.Pointer(&ret))
	sp := stringStructOf(&s)
	rp.array = sp.str
	rp.len = sp.len
	rp.cap = sp.len
	return
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testdata/print.gwrite"(%"{{.*}}/runtime/internal/runtime.Slice" %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %0, 1
// CHECK-NEXT:   %2 = icmp eq i64 %1, 0
// CHECK-NEXT:   br i1 %2, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %3 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %0, 1
// CHECK-NEXT:   br label %_llgo_3
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_4, %_llgo_2
// CHECK-NEXT:   %4 = phi i64 [ -1, %_llgo_2 ], [ %5, %_llgo_4 ]
// CHECK-NEXT:   %5 = add i64 %4, 1
// CHECK-NEXT:   %6 = icmp slt i64 %5, %3
// CHECK-NEXT:   br i1 %6, label %_llgo_4, label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_3
// CHECK-NEXT:   %7 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %0, 0
// CHECK-NEXT:   %8 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %0, 1
// CHECK-NEXT:   %9 = icmp slt i64 %5, 0
// CHECK-NEXT:   %10 = icmp uge i64 %5, %8
// CHECK-NEXT:   %11 = or i1 %10, %9
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertIndexRange"(i1 %11)
// CHECK-NEXT:   %12 = getelementptr inbounds i8, ptr %7, i64 %5
// CHECK-NEXT:   %13 = load i8, ptr %12, align 1
// CHECK-NEXT:   %14 = call i32 (ptr, ...) @printf(ptr @0, i8 %13)
// CHECK-NEXT:   br label %_llgo_3
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_3
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

func gwrite(b []byte) {
	if len(b) == 0 {
		return
	}
	for _, v := range b {
		c.Printf(c.Str("%c"), v)
	}
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testdata/print.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   call void @"{{.*}}/cl/_testdata/print.printstring"(%"{{.*}}/runtime/internal/runtime.String" { ptr @1, i64 4 })
// CHECK-NEXT:   call void @"{{.*}}/cl/_testdata/print.printnl"()
// CHECK-NEXT:   call void @"{{.*}}/cl/_testdata/print.printuint"(i64 1024)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testdata/print.printnl"()
// CHECK-NEXT:   call void @"{{.*}}/cl/_testdata/print.printhex"(i64 305441743)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testdata/print.printnl"()
// CHECK-NEXT:   call void @"{{.*}}/cl/_testdata/print.prinxor"(i64 1)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testdata/print.printnl"()
// CHECK-NEXT:   call void @"{{.*}}/cl/_testdata/print.prinsub"(i64 100)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testdata/print.printnl"()
// CHECK-NEXT:   call void @"{{.*}}/cl/_testdata/print.prinusub"(i64 -1)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testdata/print.printnl"()
// CHECK-NEXT:   call void @"{{.*}}/cl/_testdata/print.prinfsub"(double 1.001000e+02)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testdata/print.printnl"()
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store float 1.000000e+09, ptr %0, align 4
// CHECK-NEXT:   %1 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_float32, ptr undef }, ptr %0, 1
// CHECK-NEXT:   call void @"{{.*}}/cl/_testdata/print.printany"(%"{{.*}}/runtime/internal/runtime.eface" %1)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testdata/print.printnl"()
// CHECK-NEXT:   %2 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store double 2.000000e+09, ptr %2, align 8
// CHECK-NEXT:   %3 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_float64, ptr undef }, ptr %2, 1
// CHECK-NEXT:   call void @"{{.*}}/cl/_testdata/print.printany"(%"{{.*}}/runtime/internal/runtime.eface" %3)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testdata/print.printnl"()
// CHECK-NEXT:   br i1 true, label %_llgo_3, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_3
// CHECK-NEXT:   %4 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 32)
// CHECK-NEXT:   %5 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %4, i64 0
// CHECK-NEXT:   %6 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @4, i64 10 }, ptr %6, align 8
// CHECK-NEXT:   %7 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %6, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %7, ptr %5, align 8
// CHECK-NEXT:   %8 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %4, i64 1
// CHECK-NEXT:   %9 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// CHECK-NEXT:   store i1 true, ptr %9, align 1
// CHECK-NEXT:   %10 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_bool, ptr undef }, ptr %9, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %10, ptr %8, align 8
// CHECK-NEXT:   %11 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %4, 0
// CHECK-NEXT:   %12 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %11, i64 2, 1
// CHECK-NEXT:   %13 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %12, i64 2, 2
// CHECK-NEXT:   call void @"{{.*}}/cl/_testdata/print.println"(%"{{.*}}/runtime/internal/runtime.Slice" %13)
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_3, %_llgo_0
// CHECK-NEXT:   %14 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 48)
// CHECK-NEXT:   %15 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %14, i64 0
// CHECK-NEXT:   %16 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @7, i64 8 }, ptr %16, align 8
// CHECK-NEXT:   %17 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %16, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %17, ptr %15, align 8
// CHECK-NEXT:   %18 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %14, i64 1
// CHECK-NEXT:   %19 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// CHECK-NEXT:   store i1 true, ptr %19, align 1
// CHECK-NEXT:   %20 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_bool, ptr undef }, ptr %19, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %20, ptr %18, align 8
// CHECK-NEXT:   %21 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %14, i64 2
// CHECK-NEXT:   %22 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// CHECK-NEXT:   store i1 true, ptr %22, align 1
// CHECK-NEXT:   %23 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_bool, ptr undef }, ptr %22, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %23, ptr %21, align 8
// CHECK-NEXT:   %24 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %14, 0
// CHECK-NEXT:   %25 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %24, i64 3, 1
// CHECK-NEXT:   %26 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %25, i64 3, 2
// CHECK-NEXT:   call void @"{{.*}}/cl/_testdata/print.println"(%"{{.*}}/runtime/internal/runtime.Slice" %26)
// CHECK-NEXT:   %27 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 256)
// CHECK-NEXT:   %28 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %27, i64 0
// CHECK-NEXT:   %29 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// CHECK-NEXT:   store i1 true, ptr %29, align 1
// CHECK-NEXT:   %30 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_bool, ptr undef }, ptr %29, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %30, ptr %28, align 8
// CHECK-NEXT:   %31 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %27, i64 1
// CHECK-NEXT:   %32 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// CHECK-NEXT:   store i1 false, ptr %32, align 1
// CHECK-NEXT:   %33 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_bool, ptr undef }, ptr %32, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %33, ptr %31, align 8
// CHECK-NEXT:   %34 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %27, i64 2
// CHECK-NEXT:   %35 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store i32 97, ptr %35, align 4
// CHECK-NEXT:   %36 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int32, ptr undef }, ptr %35, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %36, ptr %34, align 8
// CHECK-NEXT:   %37 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %27, i64 3
// CHECK-NEXT:   %38 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store i32 65, ptr %38, align 4
// CHECK-NEXT:   %39 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int32, ptr undef }, ptr %38, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %39, ptr %37, align 8
// CHECK-NEXT:   %40 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %27, i64 4
// CHECK-NEXT:   %41 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store i32 20013, ptr %41, align 4
// CHECK-NEXT:   %42 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int32, ptr undef }, ptr %41, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %42, ptr %40, align 8
// CHECK-NEXT:   %43 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %27, i64 5
// CHECK-NEXT:   %44 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// CHECK-NEXT:   store i8 1, ptr %44, align 1
// CHECK-NEXT:   %45 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int8, ptr undef }, ptr %44, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %45, ptr %43, align 8
// CHECK-NEXT:   %46 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %27, i64 6
// CHECK-NEXT:   %47 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 2)
// CHECK-NEXT:   store i16 2, ptr %47, align 2
// CHECK-NEXT:   %48 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int16, ptr undef }, ptr %47, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %48, ptr %46, align 8
// CHECK-NEXT:   %49 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %27, i64 7
// CHECK-NEXT:   %50 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store i32 3, ptr %50, align 4
// CHECK-NEXT:   %51 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int32, ptr undef }, ptr %50, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %51, ptr %49, align 8
// CHECK-NEXT:   %52 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %27, i64 8
// CHECK-NEXT:   %53 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 4, ptr %53, align 8
// CHECK-NEXT:   %54 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int64, ptr undef }, ptr %53, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %54, ptr %52, align 8
// CHECK-NEXT:   %55 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %27, i64 9
// CHECK-NEXT:   %56 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 5, ptr %56, align 8
// CHECK-NEXT:   %57 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %56, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %57, ptr %55, align 8
// CHECK-NEXT:   %58 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %27, i64 10
// CHECK-NEXT:   %59 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// CHECK-NEXT:   store i8 1, ptr %59, align 1
// CHECK-NEXT:   %60 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint8, ptr undef }, ptr %59, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %60, ptr %58, align 8
// CHECK-NEXT:   %61 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %27, i64 11
// CHECK-NEXT:   %62 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 2)
// CHECK-NEXT:   store i16 2, ptr %62, align 2
// CHECK-NEXT:   %63 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint16, ptr undef }, ptr %62, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %63, ptr %61, align 8
// CHECK-NEXT:   %64 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %27, i64 12
// CHECK-NEXT:   %65 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store i32 3, ptr %65, align 4
// CHECK-NEXT:   %66 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint32, ptr undef }, ptr %65, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %66, ptr %64, align 8
// CHECK-NEXT:   %67 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %27, i64 13
// CHECK-NEXT:   %68 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 4, ptr %68, align 8
// CHECK-NEXT:   %69 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint64, ptr undef }, ptr %68, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %69, ptr %67, align 8
// CHECK-NEXT:   %70 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %27, i64 14
// CHECK-NEXT:   %71 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 5, ptr %71, align 8
// CHECK-NEXT:   %72 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uintptr, ptr undef }, ptr %71, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %72, ptr %70, align 8
// CHECK-NEXT:   %73 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %27, i64 15
// CHECK-NEXT:   %74 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @1, i64 4 }, ptr %74, align 8
// CHECK-NEXT:   %75 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %74, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %75, ptr %73, align 8
// CHECK-NEXT:   %76 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %27, 0
// CHECK-NEXT:   %77 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %76, i64 16, 1
// CHECK-NEXT:   %78 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %77, i64 16, 2
// CHECK-NEXT:   call void @"{{.*}}/cl/_testdata/print.println"(%"{{.*}}/runtime/internal/runtime.Slice" %78)
// CHECK-NEXT:   %79 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %80 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %79, i64 0
// CHECK-NEXT:   %81 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store { double, double } { double 1.000000e+00, double 2.000000e+00 }, ptr %81, align 8
// CHECK-NEXT:   %82 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_complex128, ptr undef }, ptr %81, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %82, ptr %80, align 8
// CHECK-NEXT:   %83 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %79, 0
// CHECK-NEXT:   %84 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %83, i64 1, 1
// CHECK-NEXT:   %85 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %84, i64 1, 2
// CHECK-NEXT:   call void @"{{.*}}/cl/_testdata/print.println"(%"{{.*}}/runtime/internal/runtime.Slice" %85)
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_0
// CHECK-NEXT:   br i1 true, label %_llgo_1, label %_llgo_2
// CHECK-NEXT: }

func main() {
	printstring("llgo")
	printnl()
	printuint(1024)
	printnl()
	printhex(0x1234abcf)
	printnl()
	prinxor(1)
	printnl()
	prinsub(100)
	printnl()
	prinusub(1<<64 - 1)
	printnl()
	prinfsub(100.1)
	printnl()
	printany(float32(1e9))
	printnl()
	printany(float64(2e9))
	printnl()
	var b bool = true
	if b == true && b != false {
		println("check bool", b)
	}
	n1 := 0b1001
	n2 := 0b0011
	println("check &^", n1&^n2 == 0b1000, n2&^n1 == 0b0010)
	println(true, false, 'a', 'A', rune('中'),
		int8(1), int16(2), int32(3), int64(4), 5,
		uint8(1), uint16(2), uint32(3), uint64(4), uintptr(5),
		"llgo")
	println(1 + 2i)
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testdata/print.prinfsub"(double %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = fneg double %0
// CHECK-NEXT:   call void @"{{.*}}/cl/_testdata/print.printfloat"(double %1)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

func prinfsub(n float64) {
	printfloat(-n)
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testdata/print.prinsub"(i64 %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = sub i64 0, %0
// CHECK-NEXT:   call void @"{{.*}}/cl/_testdata/print.printint"(i64 %1)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func prinsub(n int64) {
	printint(-n)
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testdata/print.printany"(%"{{.*}}/runtime/internal/runtime.eface" %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %0, 0
// CHECK-NEXT:   %2 = icmp eq ptr %1, @_llgo_bool
// CHECK-NEXT:   br i1 %2, label %_llgo_35, label %_llgo_36
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_34, %_llgo_85, %_llgo_32, %_llgo_30, %_llgo_28, %_llgo_26, %_llgo_24, %_llgo_22, %_llgo_20, %_llgo_18, %_llgo_16, %_llgo_14, %_llgo_12, %_llgo_10, %_llgo_8, %_llgo_6, %_llgo_4, %_llgo_2
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_37
// CHECK-NEXT:   call void @"{{.*}}/cl/_testdata/print.printbool"(i1 %53)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_37
// CHECK-NEXT:   %3 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %0, 0
// CHECK-NEXT:   %4 = icmp eq ptr %3, @_llgo_int
// CHECK-NEXT:   br i1 %4, label %_llgo_38, label %_llgo_39
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_40
// CHECK-NEXT:   call void @"{{.*}}/cl/_testdata/print.printint"(i64 %60)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_40
// CHECK-NEXT:   %5 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %0, 0
// CHECK-NEXT:   %6 = icmp eq ptr %5, @_llgo_int8
// CHECK-NEXT:   br i1 %6, label %_llgo_41, label %_llgo_42
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_43
// CHECK-NEXT:   %7 = sext i8 %67 to i64
// CHECK-NEXT:   call void @"{{.*}}/cl/_testdata/print.printint"(i64 %7)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_7:                                          ; preds = %_llgo_43
// CHECK-NEXT:   %8 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %0, 0
// CHECK-NEXT:   %9 = icmp eq ptr %8, @_llgo_int16
// CHECK-NEXT:   br i1 %9, label %_llgo_44, label %_llgo_45
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_8:                                          ; preds = %_llgo_46
// CHECK-NEXT:   %10 = sext i16 %74 to i64
// CHECK-NEXT:   call void @"{{.*}}/cl/_testdata/print.printint"(i64 %10)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_9:                                          ; preds = %_llgo_46
// CHECK-NEXT:   %11 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %0, 0
// CHECK-NEXT:   %12 = icmp eq ptr %11, @_llgo_int32
// CHECK-NEXT:   br i1 %12, label %_llgo_47, label %_llgo_48
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_10:                                         ; preds = %_llgo_49
// CHECK-NEXT:   %13 = sext i32 %81 to i64
// CHECK-NEXT:   call void @"{{.*}}/cl/_testdata/print.printint"(i64 %13)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_11:                                         ; preds = %_llgo_49
// CHECK-NEXT:   %14 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %0, 0
// CHECK-NEXT:   %15 = icmp eq ptr %14, @_llgo_int64
// CHECK-NEXT:   br i1 %15, label %_llgo_50, label %_llgo_51
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_12:                                         ; preds = %_llgo_52
// CHECK-NEXT:   call void @"{{.*}}/cl/_testdata/print.printint"(i64 %88)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_13:                                         ; preds = %_llgo_52
// CHECK-NEXT:   %16 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %0, 0
// CHECK-NEXT:   %17 = icmp eq ptr %16, @_llgo_uint
// CHECK-NEXT:   br i1 %17, label %_llgo_53, label %_llgo_54
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_14:                                         ; preds = %_llgo_55
// CHECK-NEXT:   call void @"{{.*}}/cl/_testdata/print.printuint"(i64 %95)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_15:                                         ; preds = %_llgo_55
// CHECK-NEXT:   %18 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %0, 0
// CHECK-NEXT:   %19 = icmp eq ptr %18, @_llgo_uint8
// CHECK-NEXT:   br i1 %19, label %_llgo_56, label %_llgo_57
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_16:                                         ; preds = %_llgo_58
// CHECK-NEXT:   %20 = zext i8 %102 to i64
// CHECK-NEXT:   call void @"{{.*}}/cl/_testdata/print.printuint"(i64 %20)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_17:                                         ; preds = %_llgo_58
// CHECK-NEXT:   %21 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %0, 0
// CHECK-NEXT:   %22 = icmp eq ptr %21, @_llgo_uint16
// CHECK-NEXT:   br i1 %22, label %_llgo_59, label %_llgo_60
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_18:                                         ; preds = %_llgo_61
// CHECK-NEXT:   %23 = zext i16 %109 to i64
// CHECK-NEXT:   call void @"{{.*}}/cl/_testdata/print.printuint"(i64 %23)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_19:                                         ; preds = %_llgo_61
// CHECK-NEXT:   %24 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %0, 0
// CHECK-NEXT:   %25 = icmp eq ptr %24, @_llgo_uint32
// CHECK-NEXT:   br i1 %25, label %_llgo_62, label %_llgo_63
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_20:                                         ; preds = %_llgo_64
// CHECK-NEXT:   %26 = zext i32 %116 to i64
// CHECK-NEXT:   call void @"{{.*}}/cl/_testdata/print.printuint"(i64 %26)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_21:                                         ; preds = %_llgo_64
// CHECK-NEXT:   %27 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %0, 0
// CHECK-NEXT:   %28 = icmp eq ptr %27, @_llgo_uint64
// CHECK-NEXT:   br i1 %28, label %_llgo_65, label %_llgo_66
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_22:                                         ; preds = %_llgo_67
// CHECK-NEXT:   call void @"{{.*}}/cl/_testdata/print.printuint"(i64 %123)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_23:                                         ; preds = %_llgo_67
// CHECK-NEXT:   %29 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %0, 0
// CHECK-NEXT:   %30 = icmp eq ptr %29, @_llgo_uintptr
// CHECK-NEXT:   br i1 %30, label %_llgo_68, label %_llgo_69
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_24:                                         ; preds = %_llgo_70
// CHECK-NEXT:   call void @"{{.*}}/cl/_testdata/print.printuint"(i64 %130)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_25:                                         ; preds = %_llgo_70
// CHECK-NEXT:   %31 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %0, 0
// CHECK-NEXT:   %32 = icmp eq ptr %31, @_llgo_float32
// CHECK-NEXT:   br i1 %32, label %_llgo_71, label %_llgo_72
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_26:                                         ; preds = %_llgo_73
// CHECK-NEXT:   %33 = fpext float %137 to double
// CHECK-NEXT:   call void @"{{.*}}/cl/_testdata/print.printfloat"(double %33)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_27:                                         ; preds = %_llgo_73
// CHECK-NEXT:   %34 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %0, 0
// CHECK-NEXT:   %35 = icmp eq ptr %34, @_llgo_float64
// CHECK-NEXT:   br i1 %35, label %_llgo_74, label %_llgo_75
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_28:                                         ; preds = %_llgo_76
// CHECK-NEXT:   call void @"{{.*}}/cl/_testdata/print.printfloat"(double %144)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_29:                                         ; preds = %_llgo_76
// CHECK-NEXT:   %36 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %0, 0
// CHECK-NEXT:   %37 = icmp eq ptr %36, @_llgo_complex64
// CHECK-NEXT:   br i1 %37, label %_llgo_77, label %_llgo_78
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_30:                                         ; preds = %_llgo_79
// CHECK-NEXT:   call void @"{{.*}}/cl/_testdata/print.printstring"(%"{{.*}}/runtime/internal/runtime.String" { ptr @21, i64 1 })
// CHECK-NEXT:   %38 = extractvalue { float, float } %151, 0
// CHECK-NEXT:   %39 = fpext float %38 to double
// CHECK-NEXT:   call void @"{{.*}}/cl/_testdata/print.printfloat"(double %39)
// CHECK-NEXT:   %40 = extractvalue { float, float } %151, 1
// CHECK-NEXT:   %41 = fpext float %40 to double
// CHECK-NEXT:   call void @"{{.*}}/cl/_testdata/print.printfloat"(double %41)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testdata/print.printstring"(%"{{.*}}/runtime/internal/runtime.String" { ptr @22, i64 2 })
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_31:                                         ; preds = %_llgo_79
// CHECK-NEXT:   %42 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %0, 0
// CHECK-NEXT:   %43 = icmp eq ptr %42, @_llgo_complex128
// CHECK-NEXT:   br i1 %43, label %_llgo_80, label %_llgo_81
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_32:                                         ; preds = %_llgo_82
// CHECK-NEXT:   call void @"{{.*}}/cl/_testdata/print.printstring"(%"{{.*}}/runtime/internal/runtime.String" { ptr @21, i64 1 })
// CHECK-NEXT:   %44 = extractvalue { double, double } %158, 0
// CHECK-NEXT:   call void @"{{.*}}/cl/_testdata/print.printfloat"(double %44)
// CHECK-NEXT:   %45 = extractvalue { double, double } %158, 1
// CHECK-NEXT:   call void @"{{.*}}/cl/_testdata/print.printfloat"(double %45)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testdata/print.printstring"(%"{{.*}}/runtime/internal/runtime.String" { ptr @22, i64 2 })
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_33:                                         ; preds = %_llgo_82
// CHECK-NEXT:   %46 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %0, 0
// CHECK-NEXT:   %47 = icmp eq ptr %46, @_llgo_string
// CHECK-NEXT:   br i1 %47, label %_llgo_83, label %_llgo_84
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_34:                                         ; preds = %_llgo_85
// CHECK-NEXT:   call void @"{{.*}}/cl/_testdata/print.printstring"(%"{{.*}}/runtime/internal/runtime.String" %165)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_35:                                         ; preds = %_llgo_0
// CHECK-NEXT:   %48 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %0, 1
// CHECK-NEXT:   %49 = load i1, ptr %48, align 1
// CHECK-NEXT:   %50 = insertvalue { i1, i1 } undef, i1 %49, 0
// CHECK-NEXT:   %51 = insertvalue { i1, i1 } %50, i1 true, 1
// CHECK-NEXT:   br label %_llgo_37
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_36:                                         ; preds = %_llgo_0
// CHECK-NEXT:   br label %_llgo_37
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_37:                                         ; preds = %_llgo_36, %_llgo_35
// CHECK-NEXT:   %52 = phi { i1, i1 } [ %51, %_llgo_35 ], [ zeroinitializer, %_llgo_36 ]
// CHECK-NEXT:   %53 = extractvalue { i1, i1 } %52, 0
// CHECK-NEXT:   %54 = extractvalue { i1, i1 } %52, 1
// CHECK-NEXT:   br i1 %54, label %_llgo_2, label %_llgo_3
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_38:                                         ; preds = %_llgo_3
// CHECK-NEXT:   %55 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %0, 1
// CHECK-NEXT:   %56 = load i64, ptr %55, align 8
// CHECK-NEXT:   %57 = insertvalue { i64, i1 } undef, i64 %56, 0
// CHECK-NEXT:   %58 = insertvalue { i64, i1 } %57, i1 true, 1
// CHECK-NEXT:   br label %_llgo_40
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_39:                                         ; preds = %_llgo_3
// CHECK-NEXT:   br label %_llgo_40
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_40:                                         ; preds = %_llgo_39, %_llgo_38
// CHECK-NEXT:   %59 = phi { i64, i1 } [ %58, %_llgo_38 ], [ zeroinitializer, %_llgo_39 ]
// CHECK-NEXT:   %60 = extractvalue { i64, i1 } %59, 0
// CHECK-NEXT:   %61 = extractvalue { i64, i1 } %59, 1
// CHECK-NEXT:   br i1 %61, label %_llgo_4, label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_41:                                         ; preds = %_llgo_5
// CHECK-NEXT:   %62 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %0, 1
// CHECK-NEXT:   %63 = load i8, ptr %62, align 1
// CHECK-NEXT:   %64 = insertvalue { i8, i1 } undef, i8 %63, 0
// CHECK-NEXT:   %65 = insertvalue { i8, i1 } %64, i1 true, 1
// CHECK-NEXT:   br label %_llgo_43
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_42:                                         ; preds = %_llgo_5
// CHECK-NEXT:   br label %_llgo_43
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_43:                                         ; preds = %_llgo_42, %_llgo_41
// CHECK-NEXT:   %66 = phi { i8, i1 } [ %65, %_llgo_41 ], [ zeroinitializer, %_llgo_42 ]
// CHECK-NEXT:   %67 = extractvalue { i8, i1 } %66, 0
// CHECK-NEXT:   %68 = extractvalue { i8, i1 } %66, 1
// CHECK-NEXT:   br i1 %68, label %_llgo_6, label %_llgo_7
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_44:                                         ; preds = %_llgo_7
// CHECK-NEXT:   %69 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %0, 1
// CHECK-NEXT:   %70 = load i16, ptr %69, align 2
// CHECK-NEXT:   %71 = insertvalue { i16, i1 } undef, i16 %70, 0
// CHECK-NEXT:   %72 = insertvalue { i16, i1 } %71, i1 true, 1
// CHECK-NEXT:   br label %_llgo_46
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_45:                                         ; preds = %_llgo_7
// CHECK-NEXT:   br label %_llgo_46
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_46:                                         ; preds = %_llgo_45, %_llgo_44
// CHECK-NEXT:   %73 = phi { i16, i1 } [ %72, %_llgo_44 ], [ zeroinitializer, %_llgo_45 ]
// CHECK-NEXT:   %74 = extractvalue { i16, i1 } %73, 0
// CHECK-NEXT:   %75 = extractvalue { i16, i1 } %73, 1
// CHECK-NEXT:   br i1 %75, label %_llgo_8, label %_llgo_9
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_47:                                         ; preds = %_llgo_9
// CHECK-NEXT:   %76 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %0, 1
// CHECK-NEXT:   %77 = load i32, ptr %76, align 4
// CHECK-NEXT:   %78 = insertvalue { i32, i1 } undef, i32 %77, 0
// CHECK-NEXT:   %79 = insertvalue { i32, i1 } %78, i1 true, 1
// CHECK-NEXT:   br label %_llgo_49
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_48:                                         ; preds = %_llgo_9
// CHECK-NEXT:   br label %_llgo_49
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_49:                                         ; preds = %_llgo_48, %_llgo_47
// CHECK-NEXT:   %80 = phi { i32, i1 } [ %79, %_llgo_47 ], [ zeroinitializer, %_llgo_48 ]
// CHECK-NEXT:   %81 = extractvalue { i32, i1 } %80, 0
// CHECK-NEXT:   %82 = extractvalue { i32, i1 } %80, 1
// CHECK-NEXT:   br i1 %82, label %_llgo_10, label %_llgo_11
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_50:                                         ; preds = %_llgo_11
// CHECK-NEXT:   %83 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %0, 1
// CHECK-NEXT:   %84 = load i64, ptr %83, align 8
// CHECK-NEXT:   %85 = insertvalue { i64, i1 } undef, i64 %84, 0
// CHECK-NEXT:   %86 = insertvalue { i64, i1 } %85, i1 true, 1
// CHECK-NEXT:   br label %_llgo_52
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_51:                                         ; preds = %_llgo_11
// CHECK-NEXT:   br label %_llgo_52
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_52:                                         ; preds = %_llgo_51, %_llgo_50
// CHECK-NEXT:   %87 = phi { i64, i1 } [ %86, %_llgo_50 ], [ zeroinitializer, %_llgo_51 ]
// CHECK-NEXT:   %88 = extractvalue { i64, i1 } %87, 0
// CHECK-NEXT:   %89 = extractvalue { i64, i1 } %87, 1
// CHECK-NEXT:   br i1 %89, label %_llgo_12, label %_llgo_13
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_53:                                         ; preds = %_llgo_13
// CHECK-NEXT:   %90 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %0, 1
// CHECK-NEXT:   %91 = load i64, ptr %90, align 8
// CHECK-NEXT:   %92 = insertvalue { i64, i1 } undef, i64 %91, 0
// CHECK-NEXT:   %93 = insertvalue { i64, i1 } %92, i1 true, 1
// CHECK-NEXT:   br label %_llgo_55
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_54:                                         ; preds = %_llgo_13
// CHECK-NEXT:   br label %_llgo_55
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_55:                                         ; preds = %_llgo_54, %_llgo_53
// CHECK-NEXT:   %94 = phi { i64, i1 } [ %93, %_llgo_53 ], [ zeroinitializer, %_llgo_54 ]
// CHECK-NEXT:   %95 = extractvalue { i64, i1 } %94, 0
// CHECK-NEXT:   %96 = extractvalue { i64, i1 } %94, 1
// CHECK-NEXT:   br i1 %96, label %_llgo_14, label %_llgo_15
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_56:                                         ; preds = %_llgo_15
// CHECK-NEXT:   %97 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %0, 1
// CHECK-NEXT:   %98 = load i8, ptr %97, align 1
// CHECK-NEXT:   %99 = insertvalue { i8, i1 } undef, i8 %98, 0
// CHECK-NEXT:   %100 = insertvalue { i8, i1 } %99, i1 true, 1
// CHECK-NEXT:   br label %_llgo_58
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_57:                                         ; preds = %_llgo_15
// CHECK-NEXT:   br label %_llgo_58
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_58:                                         ; preds = %_llgo_57, %_llgo_56
// CHECK-NEXT:   %101 = phi { i8, i1 } [ %100, %_llgo_56 ], [ zeroinitializer, %_llgo_57 ]
// CHECK-NEXT:   %102 = extractvalue { i8, i1 } %101, 0
// CHECK-NEXT:   %103 = extractvalue { i8, i1 } %101, 1
// CHECK-NEXT:   br i1 %103, label %_llgo_16, label %_llgo_17
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_59:                                         ; preds = %_llgo_17
// CHECK-NEXT:   %104 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %0, 1
// CHECK-NEXT:   %105 = load i16, ptr %104, align 2
// CHECK-NEXT:   %106 = insertvalue { i16, i1 } undef, i16 %105, 0
// CHECK-NEXT:   %107 = insertvalue { i16, i1 } %106, i1 true, 1
// CHECK-NEXT:   br label %_llgo_61
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_60:                                         ; preds = %_llgo_17
// CHECK-NEXT:   br label %_llgo_61
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_61:                                         ; preds = %_llgo_60, %_llgo_59
// CHECK-NEXT:   %108 = phi { i16, i1 } [ %107, %_llgo_59 ], [ zeroinitializer, %_llgo_60 ]
// CHECK-NEXT:   %109 = extractvalue { i16, i1 } %108, 0
// CHECK-NEXT:   %110 = extractvalue { i16, i1 } %108, 1
// CHECK-NEXT:   br i1 %110, label %_llgo_18, label %_llgo_19
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_62:                                         ; preds = %_llgo_19
// CHECK-NEXT:   %111 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %0, 1
// CHECK-NEXT:   %112 = load i32, ptr %111, align 4
// CHECK-NEXT:   %113 = insertvalue { i32, i1 } undef, i32 %112, 0
// CHECK-NEXT:   %114 = insertvalue { i32, i1 } %113, i1 true, 1
// CHECK-NEXT:   br label %_llgo_64
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_63:                                         ; preds = %_llgo_19
// CHECK-NEXT:   br label %_llgo_64
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_64:                                         ; preds = %_llgo_63, %_llgo_62
// CHECK-NEXT:   %115 = phi { i32, i1 } [ %114, %_llgo_62 ], [ zeroinitializer, %_llgo_63 ]
// CHECK-NEXT:   %116 = extractvalue { i32, i1 } %115, 0
// CHECK-NEXT:   %117 = extractvalue { i32, i1 } %115, 1
// CHECK-NEXT:   br i1 %117, label %_llgo_20, label %_llgo_21
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_65:                                         ; preds = %_llgo_21
// CHECK-NEXT:   %118 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %0, 1
// CHECK-NEXT:   %119 = load i64, ptr %118, align 8
// CHECK-NEXT:   %120 = insertvalue { i64, i1 } undef, i64 %119, 0
// CHECK-NEXT:   %121 = insertvalue { i64, i1 } %120, i1 true, 1
// CHECK-NEXT:   br label %_llgo_67
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_66:                                         ; preds = %_llgo_21
// CHECK-NEXT:   br label %_llgo_67
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_67:                                         ; preds = %_llgo_66, %_llgo_65
// CHECK-NEXT:   %122 = phi { i64, i1 } [ %121, %_llgo_65 ], [ zeroinitializer, %_llgo_66 ]
// CHECK-NEXT:   %123 = extractvalue { i64, i1 } %122, 0
// CHECK-NEXT:   %124 = extractvalue { i64, i1 } %122, 1
// CHECK-NEXT:   br i1 %124, label %_llgo_22, label %_llgo_23
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_68:                                         ; preds = %_llgo_23
// CHECK-NEXT:   %125 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %0, 1
// CHECK-NEXT:   %126 = load i64, ptr %125, align 8
// CHECK-NEXT:   %127 = insertvalue { i64, i1 } undef, i64 %126, 0
// CHECK-NEXT:   %128 = insertvalue { i64, i1 } %127, i1 true, 1
// CHECK-NEXT:   br label %_llgo_70
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_69:                                         ; preds = %_llgo_23
// CHECK-NEXT:   br label %_llgo_70
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_70:                                         ; preds = %_llgo_69, %_llgo_68
// CHECK-NEXT:   %129 = phi { i64, i1 } [ %128, %_llgo_68 ], [ zeroinitializer, %_llgo_69 ]
// CHECK-NEXT:   %130 = extractvalue { i64, i1 } %129, 0
// CHECK-NEXT:   %131 = extractvalue { i64, i1 } %129, 1
// CHECK-NEXT:   br i1 %131, label %_llgo_24, label %_llgo_25
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_71:                                         ; preds = %_llgo_25
// CHECK-NEXT:   %132 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %0, 1
// CHECK-NEXT:   %133 = load float, ptr %132, align 4
// CHECK-NEXT:   %134 = insertvalue { float, i1 } undef, float %133, 0
// CHECK-NEXT:   %135 = insertvalue { float, i1 } %134, i1 true, 1
// CHECK-NEXT:   br label %_llgo_73
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_72:                                         ; preds = %_llgo_25
// CHECK-NEXT:   br label %_llgo_73
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_73:                                         ; preds = %_llgo_72, %_llgo_71
// CHECK-NEXT:   %136 = phi { float, i1 } [ %135, %_llgo_71 ], [ zeroinitializer, %_llgo_72 ]
// CHECK-NEXT:   %137 = extractvalue { float, i1 } %136, 0
// CHECK-NEXT:   %138 = extractvalue { float, i1 } %136, 1
// CHECK-NEXT:   br i1 %138, label %_llgo_26, label %_llgo_27
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_74:                                         ; preds = %_llgo_27
// CHECK-NEXT:   %139 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %0, 1
// CHECK-NEXT:   %140 = load double, ptr %139, align 8
// CHECK-NEXT:   %141 = insertvalue { double, i1 } undef, double %140, 0
// CHECK-NEXT:   %142 = insertvalue { double, i1 } %141, i1 true, 1
// CHECK-NEXT:   br label %_llgo_76
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_75:                                         ; preds = %_llgo_27
// CHECK-NEXT:   br label %_llgo_76
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_76:                                         ; preds = %_llgo_75, %_llgo_74
// CHECK-NEXT:   %143 = phi { double, i1 } [ %142, %_llgo_74 ], [ zeroinitializer, %_llgo_75 ]
// CHECK-NEXT:   %144 = extractvalue { double, i1 } %143, 0
// CHECK-NEXT:   %145 = extractvalue { double, i1 } %143, 1
// CHECK-NEXT:   br i1 %145, label %_llgo_28, label %_llgo_29
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_77:                                         ; preds = %_llgo_29
// CHECK-NEXT:   %146 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %0, 1
// CHECK-NEXT:   %147 = load { float, float }, ptr %146, align 4
// CHECK-NEXT:   %148 = insertvalue { { float, float }, i1 } undef, { float, float } %147, 0
// CHECK-NEXT:   %149 = insertvalue { { float, float }, i1 } %148, i1 true, 1
// CHECK-NEXT:   br label %_llgo_79
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_78:                                         ; preds = %_llgo_29
// CHECK-NEXT:   br label %_llgo_79
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_79:                                         ; preds = %_llgo_78, %_llgo_77
// CHECK-NEXT:   %150 = phi { { float, float }, i1 } [ %149, %_llgo_77 ], [ zeroinitializer, %_llgo_78 ]
// CHECK-NEXT:   %151 = extractvalue { { float, float }, i1 } %150, 0
// CHECK-NEXT:   %152 = extractvalue { { float, float }, i1 } %150, 1
// CHECK-NEXT:   br i1 %152, label %_llgo_30, label %_llgo_31
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_80:                                         ; preds = %_llgo_31
// CHECK-NEXT:   %153 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %0, 1
// CHECK-NEXT:   %154 = load { double, double }, ptr %153, align 8
// CHECK-NEXT:   %155 = insertvalue { { double, double }, i1 } undef, { double, double } %154, 0
// CHECK-NEXT:   %156 = insertvalue { { double, double }, i1 } %155, i1 true, 1
// CHECK-NEXT:   br label %_llgo_82
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_81:                                         ; preds = %_llgo_31
// CHECK-NEXT:   br label %_llgo_82
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_82:                                         ; preds = %_llgo_81, %_llgo_80
// CHECK-NEXT:   %157 = phi { { double, double }, i1 } [ %156, %_llgo_80 ], [ zeroinitializer, %_llgo_81 ]
// CHECK-NEXT:   %158 = extractvalue { { double, double }, i1 } %157, 0
// CHECK-NEXT:   %159 = extractvalue { { double, double }, i1 } %157, 1
// CHECK-NEXT:   br i1 %159, label %_llgo_32, label %_llgo_33
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_83:                                         ; preds = %_llgo_33
// CHECK-NEXT:   %160 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %0, 1
// CHECK-NEXT:   %161 = load %"{{.*}}/runtime/internal/runtime.String", ptr %160, align 8
// CHECK-NEXT:   %162 = insertvalue { %"{{.*}}/runtime/internal/runtime.String", i1 } undef, %"{{.*}}/runtime/internal/runtime.String" %161, 0
// CHECK-NEXT:   %163 = insertvalue { %"{{.*}}/runtime/internal/runtime.String", i1 } %162, i1 true, 1
// CHECK-NEXT:   br label %_llgo_85
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_84:                                         ; preds = %_llgo_33
// CHECK-NEXT:   br label %_llgo_85
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_85:                                         ; preds = %_llgo_84, %_llgo_83
// CHECK-NEXT:   %164 = phi { %"{{.*}}/runtime/internal/runtime.String", i1 } [ %163, %_llgo_83 ], [ zeroinitializer, %_llgo_84 ]
// CHECK-NEXT:   %165 = extractvalue { %"{{.*}}/runtime/internal/runtime.String", i1 } %164, 0
// CHECK-NEXT:   %166 = extractvalue { %"{{.*}}/runtime/internal/runtime.String", i1 } %164, 1
// CHECK-NEXT:   br i1 %166, label %_llgo_34, label %_llgo_1
// CHECK-NEXT: }
func printany(v any) {
	switch v := v.(type) {
	case bool:
		printbool(v)
	case int:
		printint(int64(v))
	case int8:
		printint(int64(v))
	case int16:
		printint(int64(v))
	case int32:
		printint(int64(v))
	case int64:
		printint(int64(v))
	case uint:
		printuint(uint64(v))
	case uint8:
		printuint(uint64(v))
	case uint16:
		printuint(uint64(v))
	case uint32:
		printuint(uint64(v))
	case uint64:
		printuint(uint64(v))
	case uintptr:
		printuint(uint64(v))
	case float32:
		printfloat(float64(v))
	case float64:
		printfloat(float64(v))
	case complex64:
		printstring("(")
		printfloat(float64(real(v)))
		printfloat(float64(imag(v)))
		printstring("i)")
	case complex128:
		printstring("(")
		printfloat(real(v))
		printfloat(imag(v))
		printstring("i)")
	case string:
		printstring(v)
	}
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testdata/print.printbool"(i1 %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   br i1 %0, label %_llgo_1, label %_llgo_3
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   call void @"{{.*}}/cl/_testdata/print.printstring"(%"{{.*}}/runtime/internal/runtime.String" { ptr @23, i64 4 })
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_3, %_llgo_1
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_0
// CHECK-NEXT:   call void @"{{.*}}/cl/_testdata/print.printstring"(%"{{.*}}/runtime/internal/runtime.String" { ptr @24, i64 5 })
// CHECK-NEXT:   br label %_llgo_2
// CHECK-NEXT: }
func printbool(v bool) {
	if v {
		printstring("true")
	} else {
		printstring("false")
	}
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testdata/print.printfloat"(double %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = fcmp une double %0, %0
// CHECK-NEXT:   br i1 %1, label %_llgo_1, label %_llgo_3
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   call void @"{{.*}}/cl/_testdata/print.printstring"(%"{{.*}}/runtime/internal/runtime.String" { ptr @25, i64 3 })
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_7
// CHECK-NEXT:   call void @"{{.*}}/cl/_testdata/print.printstring"(%"{{.*}}/runtime/internal/runtime.String" { ptr @26, i64 4 })
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %2 = fadd double %0, %0
// CHECK-NEXT:   %3 = fcmp oeq double %2, %0
// CHECK-NEXT:   br i1 %3, label %_llgo_6, label %_llgo_7
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_10
// CHECK-NEXT:   call void @"{{.*}}/cl/_testdata/print.printstring"(%"{{.*}}/runtime/internal/runtime.String" { ptr @27, i64 4 })
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_7
// CHECK-NEXT:   %4 = fadd double %0, %0
// CHECK-NEXT:   %5 = fcmp oeq double %4, %0
// CHECK-NEXT:   br i1 %5, label %_llgo_9, label %_llgo_10
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_3
// CHECK-NEXT:   %6 = fcmp ogt double %0, 0.000000e+00
// CHECK-NEXT:   br label %_llgo_7
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_7:                                          ; preds = %_llgo_6, %_llgo_3
// CHECK-NEXT:   %7 = phi i1 [ false, %_llgo_3 ], [ %6, %_llgo_6 ]
// CHECK-NEXT:   br i1 %7, label %_llgo_2, label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_8:                                          ; preds = %_llgo_10
// CHECK-NEXT:   %8 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 14)
// CHECK-NEXT:   %9 = getelementptr inbounds i8, ptr %8, i64 0
// CHECK-NEXT:   store i8 43, ptr %9, align 1
// CHECK-NEXT:   %10 = fcmp oeq double %0, 0.000000e+00
// CHECK-NEXT:   br i1 %10, label %_llgo_11, label %_llgo_13
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_9:                                          ; preds = %_llgo_5
// CHECK-NEXT:   %11 = fcmp olt double %0, 0.000000e+00
// CHECK-NEXT:   br label %_llgo_10
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_10:                                         ; preds = %_llgo_9, %_llgo_5
// CHECK-NEXT:   %12 = phi i1 [ false, %_llgo_5 ], [ %11, %_llgo_9 ]
// CHECK-NEXT:   br i1 %12, label %_llgo_4, label %_llgo_8
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_11:                                         ; preds = %_llgo_8
// CHECK-NEXT:   %13 = fdiv double 1.000000e+00, %0
// CHECK-NEXT:   %14 = fcmp olt double %13, 0.000000e+00
// CHECK-NEXT:   br i1 %14, label %_llgo_14, label %_llgo_12
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_12:                                         ; preds = %_llgo_24, %_llgo_23, %_llgo_14, %_llgo_11
// CHECK-NEXT:   %15 = phi double [ %0, %_llgo_11 ], [ %36, %_llgo_23 ], [ %0, %_llgo_14 ], [ %39, %_llgo_24 ]
// CHECK-NEXT:   %16 = phi i64 [ 0, %_llgo_11 ], [ %29, %_llgo_23 ], [ 0, %_llgo_14 ], [ %38, %_llgo_24 ]
// CHECK-NEXT:   br label %_llgo_25
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_13:                                         ; preds = %_llgo_8
// CHECK-NEXT:   %17 = fcmp olt double %0, 0.000000e+00
// CHECK-NEXT:   br i1 %17, label %_llgo_15, label %_llgo_17
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_14:                                         ; preds = %_llgo_11
// CHECK-NEXT:   %18 = getelementptr inbounds i8, ptr %8, i64 0
// CHECK-NEXT:   store i8 45, ptr %18, align 1
// CHECK-NEXT:   br label %_llgo_12
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_15:                                         ; preds = %_llgo_13
// CHECK-NEXT:   %19 = fneg double %0
// CHECK-NEXT:   %20 = getelementptr inbounds i8, ptr %8, i64 0
// CHECK-NEXT:   store i8 45, ptr %20, align 1
// CHECK-NEXT:   br label %_llgo_17
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_16:                                         ; preds = %_llgo_17
// CHECK-NEXT:   %21 = add i64 %24, 1
// CHECK-NEXT:   %22 = fdiv double %23, 1.000000e+01
// CHECK-NEXT:   br label %_llgo_17
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_17:                                         ; preds = %_llgo_16, %_llgo_15, %_llgo_13
// CHECK-NEXT:   %23 = phi double [ %0, %_llgo_13 ], [ %22, %_llgo_16 ], [ %19, %_llgo_15 ]
// CHECK-NEXT:   %24 = phi i64 [ 0, %_llgo_13 ], [ %21, %_llgo_16 ], [ 0, %_llgo_15 ]
// CHECK-NEXT:   %25 = fcmp oge double %23, 1.000000e+01
// CHECK-NEXT:   br i1 %25, label %_llgo_16, label %_llgo_20
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_18:                                         ; preds = %_llgo_20
// CHECK-NEXT:   %26 = sub i64 %29, 1
// CHECK-NEXT:   %27 = fmul double %28, 1.000000e+01
// CHECK-NEXT:   br label %_llgo_20
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_19:                                         ; preds = %_llgo_20
// CHECK-NEXT:   br label %_llgo_21
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_20:                                         ; preds = %_llgo_18, %_llgo_17
// CHECK-NEXT:   %28 = phi double [ %23, %_llgo_17 ], [ %27, %_llgo_18 ]
// CHECK-NEXT:   %29 = phi i64 [ %24, %_llgo_17 ], [ %26, %_llgo_18 ]
// CHECK-NEXT:   %30 = fcmp olt double %28, 1.000000e+00
// CHECK-NEXT:   br i1 %30, label %_llgo_18, label %_llgo_19
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_21:                                         ; preds = %_llgo_22, %_llgo_19
// CHECK-NEXT:   %31 = phi double [ 5.000000e+00, %_llgo_19 ], [ %34, %_llgo_22 ]
// CHECK-NEXT:   %32 = phi i64 [ 0, %_llgo_19 ], [ %35, %_llgo_22 ]
// CHECK-NEXT:   %33 = icmp slt i64 %32, 7
// CHECK-NEXT:   br i1 %33, label %_llgo_22, label %_llgo_23
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_22:                                         ; preds = %_llgo_21
// CHECK-NEXT:   %34 = fdiv double %31, 1.000000e+01
// CHECK-NEXT:   %35 = add i64 %32, 1
// CHECK-NEXT:   br label %_llgo_21
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_23:                                         ; preds = %_llgo_21
// CHECK-NEXT:   %36 = fadd double %28, %31
// CHECK-NEXT:   %37 = fcmp oge double %36, 1.000000e+01
// CHECK-NEXT:   br i1 %37, label %_llgo_24, label %_llgo_12
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_24:                                         ; preds = %_llgo_23
// CHECK-NEXT:   %38 = add i64 %29, 1
// CHECK-NEXT:   %39 = fdiv double %36, 1.000000e+01
// CHECK-NEXT:   br label %_llgo_12
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_25:                                         ; preds = %_llgo_26, %_llgo_12
// CHECK-NEXT:   %40 = phi double [ %15, %_llgo_12 ], [ %53, %_llgo_26 ]
// CHECK-NEXT:   %41 = phi i64 [ 0, %_llgo_12 ], [ %54, %_llgo_26 ]
// CHECK-NEXT:   %42 = icmp slt i64 %41, 7
// CHECK-NEXT:   br i1 %42, label %_llgo_26, label %_llgo_27
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_26:                                         ; preds = %_llgo_25
// CHECK-NEXT:   %43 = fptosi double %40 to i64
// CHECK-NEXT:   %44 = add i64 %41, 2
// CHECK-NEXT:   %45 = add i64 %43, 48
// CHECK-NEXT:   %46 = trunc i64 %45 to i8
// CHECK-NEXT:   %47 = icmp slt i64 %44, 0
// CHECK-NEXT:   %48 = icmp uge i64 %44, 14
// CHECK-NEXT:   %49 = or i1 %48, %47
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertIndexRange"(i1 %49)
// CHECK-NEXT:   %50 = getelementptr inbounds i8, ptr %8, i64 %44
// CHECK-NEXT:   store i8 %46, ptr %50, align 1
// CHECK-NEXT:   %51 = sitofp i64 %43 to double
// CHECK-NEXT:   %52 = fsub double %40, %51
// CHECK-NEXT:   %53 = fmul double %52, 1.000000e+01
// CHECK-NEXT:   %54 = add i64 %41, 1
// CHECK-NEXT:   br label %_llgo_25
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_27:                                         ; preds = %_llgo_25
// CHECK-NEXT:   %55 = getelementptr inbounds i8, ptr %8, i64 2
// CHECK-NEXT:   %56 = load i8, ptr %55, align 1
// CHECK-NEXT:   %57 = getelementptr inbounds i8, ptr %8, i64 1
// CHECK-NEXT:   store i8 %56, ptr %57, align 1
// CHECK-NEXT:   %58 = getelementptr inbounds i8, ptr %8, i64 2
// CHECK-NEXT:   store i8 46, ptr %58, align 1
// CHECK-NEXT:   %59 = getelementptr inbounds i8, ptr %8, i64 9
// CHECK-NEXT:   store i8 101, ptr %59, align 1
// CHECK-NEXT:   %60 = getelementptr inbounds i8, ptr %8, i64 10
// CHECK-NEXT:   store i8 43, ptr %60, align 1
// CHECK-NEXT:   %61 = icmp slt i64 %16, 0
// CHECK-NEXT:   br i1 %61, label %_llgo_28, label %_llgo_29
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_28:                                         ; preds = %_llgo_27
// CHECK-NEXT:   %62 = sub i64 0, %16
// CHECK-NEXT:   %63 = getelementptr inbounds i8, ptr %8, i64 10
// CHECK-NEXT:   store i8 45, ptr %63, align 1
// CHECK-NEXT:   br label %_llgo_29
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_29:                                         ; preds = %_llgo_28, %_llgo_27
// CHECK-NEXT:   %64 = phi i64 [ %16, %_llgo_27 ], [ %62, %_llgo_28 ]
// CHECK-NEXT:   %65 = sdiv i64 %64, 100
// CHECK-NEXT:   %66 = trunc i64 %65 to i8
// CHECK-NEXT:   %67 = add i8 %66, 48
// CHECK-NEXT:   %68 = getelementptr inbounds i8, ptr %8, i64 11
// CHECK-NEXT:   store i8 %67, ptr %68, align 1
// CHECK-NEXT:   %69 = sdiv i64 %64, 10
// CHECK-NEXT:   %70 = trunc i64 %69 to i8
// CHECK-NEXT:   %71 = urem i8 %70, 10
// CHECK-NEXT:   %72 = add i8 %71, 48
// CHECK-NEXT:   %73 = getelementptr inbounds i8, ptr %8, i64 12
// CHECK-NEXT:   store i8 %72, ptr %73, align 1
// CHECK-NEXT:   %74 = srem i64 %64, 10
// CHECK-NEXT:   %75 = trunc i64 %74 to i8
// CHECK-NEXT:   %76 = add i8 %75, 48
// CHECK-NEXT:   %77 = getelementptr inbounds i8, ptr %8, i64 13
// CHECK-NEXT:   store i8 %76, ptr %77, align 1
// CHECK-NEXT:   %78 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %8, 0
// CHECK-NEXT:   %79 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %78, i64 14, 1
// CHECK-NEXT:   %80 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %79, i64 14, 2
// CHECK-NEXT:   call void @"{{.*}}/cl/_testdata/print.gwrite"(%"{{.*}}/runtime/internal/runtime.Slice" %80)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func printfloat(v float64) {
	switch {
	case v != v:
		printstring("NaN")
		return
	case v+v == v && v > 0:
		printstring("+Inf")
		return
	case v+v == v && v < 0:
		printstring("-Inf")
		return
	}

	const n = 7 // digits printed
	var buf [n + 7]byte
	buf[0] = '+'
	e := 0 // exp
	if v == 0 {
		if 1/v < 0 {
			buf[0] = '-'
		}
	} else {
		if v < 0 {
			v = -v
			buf[0] = '-'
		}

		// normalize
		for v >= 10 {
			e++
			v /= 10
		}
		for v < 1 {
			e--
			v *= 10
		}

		// round
		h := 5.0
		for i := 0; i < n; i++ {
			h /= 10
		}
		v += h
		if v >= 10 {
			e++
			v /= 10
		}
	}

	// format +d.dddd+edd
	for i := 0; i < n; i++ {
		s := int(v)
		buf[i+2] = byte(s + '0')
		v -= float64(s)
		v *= 10
	}
	buf[1] = buf[2]
	buf[2] = '.'

	buf[n+2] = 'e'
	buf[n+3] = '+'
	if e < 0 {
		e = -e
		buf[n+3] = '-'
	}

	buf[n+4] = byte(e/100) + '0'
	buf[n+5] = byte(e/10)%10 + '0'
	buf[n+6] = byte(e%10) + '0'
	gwrite(buf[:])
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testdata/print.printhex"(i64 %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 100)
// CHECK-NEXT:   br label %_llgo_3
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_3
// CHECK-NEXT:   %2 = urem i64 %22, 16
// CHECK-NEXT:   %3 = icmp uge i64 %2, 16
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertIndexRange"(i1 %3)
// CHECK-NEXT:   %4 = getelementptr inbounds i8, ptr @28, i64 %2
// CHECK-NEXT:   %5 = load i8, ptr %4, align 1
// CHECK-NEXT:   %6 = icmp slt i64 %23, 0
// CHECK-NEXT:   %7 = icmp uge i64 %23, 100
// CHECK-NEXT:   %8 = or i1 %7, %6
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertIndexRange"(i1 %8)
// CHECK-NEXT:   %9 = getelementptr inbounds i8, ptr %1, i64 %23
// CHECK-NEXT:   store i8 %5, ptr %9, align 1
// CHECK-NEXT:   %10 = icmp ult i64 %22, 16
// CHECK-NEXT:   br i1 %10, label %_llgo_5, label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_5, %_llgo_3
// CHECK-NEXT:   %11 = sub i64 %23, 1
// CHECK-NEXT:   %12 = icmp slt i64 %11, 0
// CHECK-NEXT:   %13 = icmp uge i64 %11, 100
// CHECK-NEXT:   %14 = or i1 %13, %12
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertIndexRange"(i1 %14)
// CHECK-NEXT:   %15 = getelementptr inbounds i8, ptr %1, i64 %11
// CHECK-NEXT:   store i8 120, ptr %15, align 1
// CHECK-NEXT:   %16 = sub i64 %11, 1
// CHECK-NEXT:   %17 = icmp slt i64 %16, 0
// CHECK-NEXT:   %18 = icmp uge i64 %16, 100
// CHECK-NEXT:   %19 = or i1 %18, %17
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertIndexRange"(i1 %19)
// CHECK-NEXT:   %20 = getelementptr inbounds i8, ptr %1, i64 %16
// CHECK-NEXT:   store i8 48, ptr %20, align 1
// CHECK-NEXT:   %21 = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/internal/runtime.NewSlice3"(ptr %1, i64 1, i64 100, i64 %16, i64 100, i64 100)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testdata/print.gwrite"(%"{{.*}}/runtime/internal/runtime.Slice" %21)
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_4, %_llgo_0
// CHECK-NEXT:   %22 = phi i64 [ %0, %_llgo_0 ], [ %25, %_llgo_4 ]
// CHECK-NEXT:   %23 = phi i64 [ 99, %_llgo_0 ], [ %26, %_llgo_4 ]
// CHECK-NEXT:   %24 = icmp sgt i64 %23, 0
// CHECK-NEXT:   br i1 %24, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_5, %_llgo_1
// CHECK-NEXT:   %25 = udiv i64 %22, 16
// CHECK-NEXT:   %26 = sub i64 %23, 1
// CHECK-NEXT:   br label %_llgo_3
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_1
// CHECK-NEXT:   %27 = sub i64 100, %23
// CHECK-NEXT:   %28 = load i64, ptr @"{{.*}}/cl/_testdata/print.minhexdigits", align 8
// CHECK-NEXT:   %29 = icmp sge i64 %27, %28
// CHECK-NEXT:   br i1 %29, label %_llgo_2, label %_llgo_4
// CHECK-NEXT: }
func printhex(v uint64) {
	const dig = "0123456789abcdef"
	var buf [100]byte
	i := len(buf)
	for i--; i > 0; i-- {
		buf[i] = dig[v%16]
		if v < 16 && len(buf)-i >= minhexdigits {
			break
		}
		v /= 16
	}
	i--
	buf[i] = 'x'
	i--
	buf[i] = '0'
	gwrite(buf[i:])
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testdata/print.printint"(i64 %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = icmp slt i64 %0, 0
// CHECK-NEXT:   br i1 %1, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   call void @"{{.*}}/cl/_testdata/print.printstring"(%"{{.*}}/runtime/internal/runtime.String" { ptr @29, i64 1 })
// CHECK-NEXT:   %2 = sub i64 0, %0
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   %3 = phi i64 [ %0, %_llgo_0 ], [ %2, %_llgo_1 ]
// CHECK-NEXT:   call void @"{{.*}}/cl/_testdata/print.printuint"(i64 %3)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func printint(v int64) {
	if v < 0 {
		printstring("-")
		v = -v
	}
	printuint(uint64(v))
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testdata/print.println"(%"{{.*}}/runtime/internal/runtime.Slice" %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %0, 1
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_5, %_llgo_0
// CHECK-NEXT:   %2 = phi i64 [ -1, %_llgo_0 ], [ %3, %_llgo_5 ]
// CHECK-NEXT:   %3 = add i64 %2, 1
// CHECK-NEXT:   %4 = icmp slt i64 %3, %1
// CHECK-NEXT:   br i1 %4, label %_llgo_2, label %_llgo_3
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1
// CHECK-NEXT:   %5 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %0, 0
// CHECK-NEXT:   %6 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %0, 1
// CHECK-NEXT:   %7 = icmp slt i64 %3, 0
// CHECK-NEXT:   %8 = icmp uge i64 %3, %6
// CHECK-NEXT:   %9 = or i1 %8, %7
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertIndexRange"(i1 %9)
// CHECK-NEXT:   %10 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %5, i64 %3
// CHECK-NEXT:   %11 = load %"{{.*}}/runtime/internal/runtime.eface", ptr %10, align 8
// CHECK-NEXT:   %12 = icmp ne i64 %3, 0
// CHECK-NEXT:   br i1 %12, label %_llgo_4, label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_1
// CHECK-NEXT:   call void @"{{.*}}/cl/_testdata/print.printnl"()
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_2
// CHECK-NEXT:   call void @"{{.*}}/cl/_testdata/print.printstring"(%"{{.*}}/runtime/internal/runtime.String" { ptr @30, i64 1 })
// CHECK-NEXT:   br label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_4, %_llgo_2
// CHECK-NEXT:   call void @"{{.*}}/cl/_testdata/print.printany"(%"{{.*}}/runtime/internal/runtime.eface" %11)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-NEXT: }
func println(args ...any) {
	for i, v := range args {
		if i != 0 {
			printstring(" ")
		}
		printany(v)
	}
	printnl()
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testdata/print.printnl"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   call void @"{{.*}}/cl/_testdata/print.printstring"(%"{{.*}}/runtime/internal/runtime.String" { ptr @31, i64 1 })
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func printnl() {
	printstring("\n")
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testdata/print.printsp"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   call void @"{{.*}}/cl/_testdata/print.printstring"(%"{{.*}}/runtime/internal/runtime.String" { ptr @30, i64 1 })
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func printsp() {
	printstring(" ")
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testdata/print.printstring"(%"{{.*}}/runtime/internal/runtime.String" %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/cl/_testdata/print.bytes"(%"{{.*}}/runtime/internal/runtime.String" %0)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testdata/print.gwrite"(%"{{.*}}/runtime/internal/runtime.Slice" %1)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func printstring(s string) {
	gwrite(bytes(s))
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testdata/print.printuint"(i64 %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 100)
// CHECK-NEXT:   br label %_llgo_3
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_3
// CHECK-NEXT:   %2 = urem i64 %11, 10
// CHECK-NEXT:   %3 = add i64 %2, 48
// CHECK-NEXT:   %4 = trunc i64 %3 to i8
// CHECK-NEXT:   %5 = icmp slt i64 %12, 0
// CHECK-NEXT:   %6 = icmp uge i64 %12, 100
// CHECK-NEXT:   %7 = or i1 %6, %5
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertIndexRange"(i1 %7)
// CHECK-NEXT:   %8 = getelementptr inbounds i8, ptr %1, i64 %12
// CHECK-NEXT:   store i8 %4, ptr %8, align 1
// CHECK-NEXT:   %9 = icmp ult i64 %11, 10
// CHECK-NEXT:   br i1 %9, label %_llgo_2, label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_3
// CHECK-NEXT:   %10 = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/internal/runtime.NewSlice3"(ptr %1, i64 1, i64 100, i64 %12, i64 100, i64 100)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testdata/print.gwrite"(%"{{.*}}/runtime/internal/runtime.Slice" %10)
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_4, %_llgo_0
// CHECK-NEXT:   %11 = phi i64 [ %0, %_llgo_0 ], [ %14, %_llgo_4 ]
// CHECK-NEXT:   %12 = phi i64 [ 99, %_llgo_0 ], [ %15, %_llgo_4 ]
// CHECK-NEXT:   %13 = icmp sgt i64 %12, 0
// CHECK-NEXT:   br i1 %13, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_1
// CHECK-NEXT:   %14 = udiv i64 %11, 10
// CHECK-NEXT:   %15 = sub i64 %12, 1
// CHECK-NEXT:   br label %_llgo_3
// CHECK-NEXT: }
func printuint(v uint64) {
	var buf [100]byte
	i := len(buf)
	for i--; i > 0; i-- {
		buf[i] = byte(v%10 + '0')
		if v < 10 {
			break
		}
		v /= 10
	}
	gwrite(buf[i:])
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testdata/print.prinusub"(i64 %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = sub i64 0, %0
// CHECK-NEXT:   call void @"{{.*}}/cl/_testdata/print.printuint"(i64 %1)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func prinusub(n uint64) {
	printuint(-n)
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testdata/print.prinxor"(i64 %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = xor i64 %0, -1
// CHECK-NEXT:   call void @"{{.*}}/cl/_testdata/print.printint"(i64 %1)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

func prinxor(n int64) {
	printint(^n)
}

// CHECK-LABEL: define ptr @"{{.*}}/cl/_testdata/print.stringStructOf"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   ret ptr %0
// CHECK-NEXT: }

func stringStructOf(sp *string) *stringStruct {
	return (*stringStruct)(unsafe.Pointer(sp))
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

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequal8"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.memequal8"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequal16"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.memequal16"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequal64"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.memequal64"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.c128equal"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.c128equal"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.c64equal"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.c64equal"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }
