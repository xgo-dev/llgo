// LITTEST
package main

import (
	"unsafe"

	"github.com/goplus/lib/c"
)

//llgo:link (*T).Printf C.printf

// CHECK-LINE: @0 = private unnamed_addr constant [9 x i8] c"%s (%d)\0A\00", align 1
// CHECK-LINE: @1 = private unnamed_addr constant [6 x i8] c"hello\00", align 1
// CHECK-LINE: @2 = private unnamed_addr constant [9 x i8] c"(%d) %s\0A\00", align 1
// CHECK-LINE: @3 = private unnamed_addr constant [6 x i8] c"world\00", align 1
// CHECK-LINE: @18 = private unnamed_addr constant [12 x i8] c"%s (%d,%d)\0A\00", align 1
// CHECK-LINE: @19 = private unnamed_addr constant [5 x i8] c"ifmt\00", align 1
// CHECK-LINE: @20 = private unnamed_addr constant [5 x i8] c"error", align 1

func (*T) Printf(__llgo_va_list ...any) c.Int { return 0 }

type T c.Char

//go:linkname Printf C.printf
func Printf(format *c.Char, __llgo_va_list ...any) c.Int

type CFmt struct {
	*T
}

func (f *CFmt) SetFormat(fmt *c.Char) {
	f.T = (*T)(unsafe.Pointer(fmt))
}

type IFmt interface {
	SetFormat(fmt *c.Char)
	Printf(__llgo_va_list ...any) c.Int
}

func main() {
	cfmt := &CFmt{}
	cfmt.SetFormat(c.Str("%s (%d)\n"))
	cfmt.Printf(c.Str("hello"), 100)
	cfmt.SetFormat(c.Str("(%d) %s\n"))
	cfmt.Printf(200, c.Str("world"))

	var i any = &CFmt{}
	ifmt, ok := i.(IFmt)
	if !ok {
		panic("error")
	}
	ifmt.SetFormat(c.Str("%s (%d,%d)\n"))
	ifmt.Printf(c.Str("ifmt"), 100, 200)
}

// CHECK-LABEL: define i32 @"{{.*}}/cl/_testrt/vamethod.CFmt.Printf"(%"{{.*}}/cl/_testrt/vamethod.CFmt" %0, ...){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = alloca %"{{.*}}/cl/_testrt/vamethod.CFmt", align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %1, i8 0, i64 8, i1 false)
// CHECK-NEXT:   store %"{{.*}}/cl/_testrt/vamethod.CFmt" %0, ptr %1, align 8
// CHECK-NEXT:   %2 = icmp eq ptr %1, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %2)
// CHECK-NEXT:   %3 = getelementptr inbounds %"{{.*}}/cl/_testrt/vamethod.CFmt", ptr %1, i32 0, i32 0
// CHECK-NEXT:   %4 = load ptr, ptr %3, align 8
// CHECK-NEXT:   %5 = call i32 (ptr, ...) @"{{.*}}/cl/_testrt/vamethod.(*T).Printf"(ptr %4)
// CHECK-NEXT:   ret i32 %5
// CHECK-NEXT: }

// CHECK-LABEL: define i32 @"{{.*}}/cl/_testrt/vamethod.(*CFmt).Printf"(ptr %0, ...){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %1)
// CHECK-NEXT:   %2 = getelementptr inbounds %"{{.*}}/cl/_testrt/vamethod.CFmt", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call i32 (ptr, ...) @"{{.*}}/cl/_testrt/vamethod.(*T).Printf"(ptr %3)
// CHECK-NEXT:   ret i32 %4
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/vamethod.(*CFmt).SetFormat"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %2)
// CHECK-NEXT:   %3 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %3)
// CHECK-NEXT:   %4 = getelementptr inbounds %"{{.*}}/cl/_testrt/vamethod.CFmt", ptr %0, i32 0, i32 0
// CHECK-NEXT:   store ptr %1, ptr %4, align 8
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define i32 @"{{.*}}/cl/_testrt/vamethod.(*T).Printf"(ptr %0, ...){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   ret i32 0
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/vamethod.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testrt/vamethod.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testrt/vamethod.init$guard", align 1
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/vamethod.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/vamethod.(*CFmt).SetFormat"(ptr %0, ptr @0)
// CHECK-NEXT:   %1 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %1)
// CHECK-NEXT:   %2 = getelementptr inbounds %"{{.*}}/cl/_testrt/vamethod.CFmt", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call i32 (ptr, ...) @"{{.*}}/cl/_testrt/vamethod.(*T).Printf"(ptr %3, ptr @1, i64 100)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/vamethod.(*CFmt).SetFormat"(ptr %0, ptr @2)
// CHECK-NEXT:   %5 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %5)
// CHECK-NEXT:   %6 = getelementptr inbounds %"{{.*}}/cl/_testrt/vamethod.CFmt", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %7 = load ptr, ptr %6, align 8
// CHECK-NEXT:   %8 = call i32 (ptr, ...) @"{{.*}}/cl/_testrt/vamethod.(*T).Printf"(ptr %7, i64 200, ptr @3)
// CHECK-NEXT:   %9 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %10 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"*_llgo_{{.*}}/cl/_testrt/vamethod.CFmt", ptr undef }, ptr %9, 1
// CHECK-NEXT:   %11 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %10, 0
// CHECK-NEXT:   %12 = call i1 @"{{.*}}/runtime/internal/runtime.Implements"(ptr @"_llgo_{{.*}}/cl/_testrt/vamethod.IFmt", ptr %11)
// CHECK-NEXT:   br i1 %12, label %_llgo_3, label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_5
// CHECK-NEXT:   %13 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @20, i64 5 }, ptr %13, align 8
// CHECK-NEXT:   %14 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %13, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %14)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_5
// CHECK-NEXT:   %15 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %39)
// CHECK-NEXT:   %16 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %39, 0
// CHECK-NEXT:   %17 = getelementptr ptr, ptr %16, i64 4
// CHECK-NEXT:   %18 = load ptr, ptr %17, align 8
// CHECK-NEXT:   %19 = insertvalue { ptr, ptr } undef, ptr %18, 0
// CHECK-NEXT:   %20 = insertvalue { ptr, ptr } %19, ptr %15, 1
// CHECK-NEXT:   %21 = extractvalue { ptr, ptr } %20, 1
// CHECK-NEXT:   %22 = extractvalue { ptr, ptr } %20, 0
// CHECK-NEXT:   call void %22(ptr %21, ptr @18)
// CHECK-NEXT:   %23 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %39)
// CHECK-NEXT:   %24 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %39, 0
// CHECK-NEXT:   %25 = getelementptr ptr, ptr %24, i64 3
// CHECK-NEXT:   %26 = load ptr, ptr %25, align 8
// CHECK-NEXT:   %27 = insertvalue { ptr, ptr } undef, ptr %26, 0
// CHECK-NEXT:   %28 = insertvalue { ptr, ptr } %27, ptr %23, 1
// CHECK-NEXT:   %29 = extractvalue { ptr, ptr } %28, 1
// CHECK-NEXT:   %30 = extractvalue { ptr, ptr } %28, 0
// CHECK-NEXT:   %31 = call i32 (ptr, ...) %30(ptr %29, ptr @19, i64 100, i64 200)
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %32 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %10, 1
// CHECK-NEXT:   %33 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface$a85zs5wWQQoPIERm_en8plssh4spdIeeXZPC-E0TDh0", ptr %11)
// CHECK-NEXT:   %34 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %33, 0
// CHECK-NEXT:   %35 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %34, ptr %32, 1
// CHECK-NEXT:   %36 = insertvalue { %"{{.*}}/runtime/internal/runtime.iface", i1 } undef, %"{{.*}}/runtime/internal/runtime.iface" %35, 0
// CHECK-NEXT:   %37 = insertvalue { %"{{.*}}/runtime/internal/runtime.iface", i1 } %36, i1 true, 1
// CHECK-NEXT:   br label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_0
// CHECK-NEXT:   br label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_4, %_llgo_3
// CHECK-NEXT:   %38 = phi { %"{{.*}}/runtime/internal/runtime.iface", i1 } [ %37, %_llgo_3 ], [ zeroinitializer, %_llgo_4 ]
// CHECK-NEXT:   %39 = extractvalue { %"{{.*}}/runtime/internal/runtime.iface", i1 } %38, 0
// CHECK-NEXT:   %40 = extractvalue { %"{{.*}}/runtime/internal/runtime.iface", i1 } %38, 1
// CHECK-NEXT:   br i1 %40, label %_llgo_2, label %_llgo_1
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequal8"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.memequal8"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i32 @"__llgo_stub.{{.*}}/cl/_testrt/vamethod.(*T).Printf"(ptr %0, ptr %1, ...){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i32 (ptr, ...) @"{{.*}}/cl/_testrt/vamethod.(*T).Printf"(ptr %1)
// CHECK-NEXT:   ret i32 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.nilinterequal"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.nilinterequal"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i32 @"__llgo_stub.{{.*}}/cl/_testrt/vamethod.CFmt.Printf"(ptr %0, %"{{.*}}/cl/_testrt/vamethod.CFmt" %1, ...){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i32 (%"{{.*}}/cl/_testrt/vamethod.CFmt", ...) @"{{.*}}/cl/_testrt/vamethod.CFmt.Printf"(%"{{.*}}/cl/_testrt/vamethod.CFmt" %1)
// CHECK-NEXT:   ret i32 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i32 @"__llgo_stub.{{.*}}/cl/_testrt/vamethod.(*CFmt).Printf"(ptr %0, ptr %1, ...){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i32 (ptr, ...) @"{{.*}}/cl/_testrt/vamethod.(*CFmt).Printf"(ptr %1)
// CHECK-NEXT:   ret i32 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce void @"__llgo_stub.{{.*}}/cl/_testrt/vamethod.(*CFmt).SetFormat"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   tail call void @"{{.*}}/cl/_testrt/vamethod.(*CFmt).SetFormat"(ptr %1, ptr %2)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.interequal"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.interequal"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }
