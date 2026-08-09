// LITTEST
package main

import (
	"unsafe"

	"github.com/goplus/lib/c"
)

// CHECK-LABEL: define i32 @main.CFmt.Printf(%main.CFmt %0, ...){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = alloca %main.CFmt, align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %1, i8 0, i64 8, i1 false)
// CHECK-NEXT:   store %main.CFmt %0, ptr %1, align 8
// CHECK-NEXT:   %2 = getelementptr inbounds %main.CFmt, ptr %1, i32 0, i32 0
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call i32 (ptr, ...) @printf(ptr %3)
// CHECK-NEXT:   ret i32 %4
// CHECK-NEXT: }

// CHECK-LABEL: define i32 @"main.(*CFmt).Printf"(ptr %0, ...){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds %main.CFmt, ptr %0, i32 0, i32 0
// CHECK-NEXT:   %2 = load ptr, ptr %1, align 8
// CHECK-NEXT:   %3 = call i32 (ptr, ...) @printf(ptr %2)
// CHECK-NEXT:   ret i32 %3
// CHECK-NEXT: }

// CHECK-LABEL: define void @"main.(*CFmt).SetFormat"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = getelementptr inbounds %main.CFmt, ptr %0, i32 0, i32 0
// CHECK-NEXT:   store ptr %1, ptr %2, align 8
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

//llgo:link (*T).Printf C.printf
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

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   call void @"main.(*CFmt).SetFormat"(ptr %0, ptr @0)
// CHECK-NEXT:   %1 = getelementptr inbounds %main.CFmt, ptr %0, i32 0, i32 0
// CHECK-NEXT:   %2 = load ptr, ptr %1, align 8
// CHECK-NEXT:   %3 = call i32 (ptr, ...) @printf(ptr %2, ptr @1, i64 100)
// CHECK-NEXT:   call void @"main.(*CFmt).SetFormat"(ptr %0, ptr @2)
// CHECK-NEXT:   %4 = getelementptr inbounds %main.CFmt, ptr %0, i32 0, i32 0
// CHECK-NEXT:   %5 = load ptr, ptr %4, align 8
// CHECK-NEXT:   %6 = call i32 (ptr, ...) @printf(ptr %5, i64 200, ptr @3)
// CHECK-NEXT:   %7 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %8 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"*_llgo_main.CFmt", ptr undef }, ptr %7, 1
// CHECK-NEXT:   %9 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %8, 0
// CHECK-NEXT:   %10 = call i1 @"{{.*}}/runtime/internal/runtime.Implements"(ptr @_llgo_main.IFmt, ptr %9)
// CHECK-NEXT:   br i1 %10, label %_llgo_3, label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_5
// CHECK-NEXT:   %11 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @20, i64 5 }, ptr %11, align 8
// CHECK-NEXT:   %12 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %11, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %12)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_5
// CHECK-NEXT:   %13 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %37)
// CHECK-NEXT:   %14 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %37, 0
// CHECK-NEXT:   %15 = getelementptr ptr, ptr %14, i64 4
// CHECK-NEXT:   %16 = load ptr, ptr %15, align 8
// CHECK-NEXT:   %17 = insertvalue { ptr, ptr } undef, ptr %16, 0
// CHECK-NEXT:   %18 = insertvalue { ptr, ptr } %17, ptr %13, 1
// CHECK-NEXT:   %19 = extractvalue { ptr, ptr } %18, 1
// CHECK-NEXT:   %20 = extractvalue { ptr, ptr } %18, 0
// CHECK-NEXT:   call void %20(ptr %19, ptr @18)
// CHECK-NEXT:   %21 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %37)
// CHECK-NEXT:   %22 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %37, 0
// CHECK-NEXT:   %23 = getelementptr ptr, ptr %22, i64 3
// CHECK-NEXT:   %24 = load ptr, ptr %23, align 8
// CHECK-NEXT:   %25 = insertvalue { ptr, ptr } undef, ptr %24, 0
// CHECK-NEXT:   %26 = insertvalue { ptr, ptr } %25, ptr %21, 1
// CHECK-NEXT:   %27 = extractvalue { ptr, ptr } %26, 1
// CHECK-NEXT:   %28 = extractvalue { ptr, ptr } %26, 0
// CHECK-NEXT:   %29 = call i32 (ptr, ...) %28(ptr %27, ptr @19, i64 100, i64 200)
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %30 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %8, 1
// CHECK-NEXT:   %31 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface$a85zs5wWQQoPIERm_en8plssh4spdIeeXZPC-E0TDh0", ptr %9)
// CHECK-NEXT:   %32 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %31, 0
// CHECK-NEXT:   %33 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %32, ptr %30, 1
// CHECK-NEXT:   %34 = insertvalue { %"{{.*}}/runtime/internal/runtime.iface", i1 } undef, %"{{.*}}/runtime/internal/runtime.iface" %33, 0
// CHECK-NEXT:   %35 = insertvalue { %"{{.*}}/runtime/internal/runtime.iface", i1 } %34, i1 true, 1
// CHECK-NEXT:   br label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_0
// CHECK-NEXT:   br label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_4, %_llgo_3
// CHECK-NEXT:   %36 = phi { %"{{.*}}/runtime/internal/runtime.iface", i1 } [ %35, %_llgo_3 ], [ zeroinitializer, %_llgo_4 ]
// CHECK-NEXT:   %37 = extractvalue { %"{{.*}}/runtime/internal/runtime.iface", i1 } %36, 0
// CHECK-NEXT:   %38 = extractvalue { %"{{.*}}/runtime/internal/runtime.iface", i1 } %36, 1
// CHECK-NEXT:   br i1 %38, label %_llgo_2, label %_llgo_1
// CHECK-NEXT: }
// ESCAPE-LABEL: define void @main.main(){{.*}} {
// ESCAPE-NEXT: _llgo_0:
// ESCAPE-NEXT:   %.stack = alloca i8, i64 8, align 8
// ESCAPE-NEXT:   call void @llvm.memset.p0.i64(ptr %.stack, i8 0, i64 8, i1 false)
// ESCAPE-NEXT:   call void @"main.(*CFmt).SetFormat"(ptr %.stack, ptr @0)
// ESCAPE-NEXT:   %0 = getelementptr inbounds %main.CFmt, ptr %.stack, i32 0, i32 0
// ESCAPE-NEXT:   %1 = load ptr, ptr %0, align 8
// ESCAPE-NEXT:   %2 = call i32 (ptr, ...) @printf(ptr %1, ptr @1, i64 100)
// ESCAPE-NEXT:   call void @"main.(*CFmt).SetFormat"(ptr %.stack, ptr @2)
// ESCAPE-NEXT:   %3 = getelementptr inbounds %main.CFmt, ptr %.stack, i32 0, i32 0
// ESCAPE-NEXT:   %4 = load ptr, ptr %3, align 8
// ESCAPE-NEXT:   %5 = call i32 (ptr, ...) @printf(ptr %4, i64 200, ptr @3)
// ESCAPE-NEXT:   %6 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// ESCAPE-NEXT:   %7 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"*_llgo_main.CFmt", ptr undef }, ptr %6, 1
// ESCAPE-NEXT:   %8 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %7, 0
// ESCAPE-NEXT:   %9 = call i1 @"{{.*}}/runtime/internal/runtime.Implements"(ptr @_llgo_main.IFmt, ptr %8)
// ESCAPE-NEXT:   br i1 %9, label %_llgo_3, label %_llgo_4
// ESCAPE-EMPTY:
// ESCAPE-NEXT: _llgo_1:                                          ; preds = %_llgo_5
// ESCAPE-NEXT:   %10 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// ESCAPE-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @20, i64 5 }, ptr %10, align 8
// ESCAPE-NEXT:   %11 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %10, 1
// ESCAPE-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %11)
// ESCAPE-NEXT:   unreachable
// ESCAPE-EMPTY:
// ESCAPE-NEXT: _llgo_2:                                          ; preds = %_llgo_5
// ESCAPE-NEXT:   %12 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %36)
// ESCAPE-NEXT:   %13 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %36, 0
// ESCAPE-NEXT:   %14 = getelementptr ptr, ptr %13, i64 4
// ESCAPE-NEXT:   %15 = load ptr, ptr %14, align 8
// ESCAPE-NEXT:   %16 = insertvalue { ptr, ptr } undef, ptr %15, 0
// ESCAPE-NEXT:   %17 = insertvalue { ptr, ptr } %16, ptr %12, 1
// ESCAPE-NEXT:   %18 = extractvalue { ptr, ptr } %17, 1
// ESCAPE-NEXT:   %19 = extractvalue { ptr, ptr } %17, 0
// ESCAPE-NEXT:   call void %19(ptr %18, ptr @18)
// ESCAPE-NEXT:   %20 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %36)
// ESCAPE-NEXT:   %21 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %36, 0
// ESCAPE-NEXT:   %22 = getelementptr ptr, ptr %21, i64 3
// ESCAPE-NEXT:   %23 = load ptr, ptr %22, align 8
// ESCAPE-NEXT:   %24 = insertvalue { ptr, ptr } undef, ptr %23, 0
// ESCAPE-NEXT:   %25 = insertvalue { ptr, ptr } %24, ptr %20, 1
// ESCAPE-NEXT:   %26 = extractvalue { ptr, ptr } %25, 1
// ESCAPE-NEXT:   %27 = extractvalue { ptr, ptr } %25, 0
// ESCAPE-NEXT:   %28 = call i32 (ptr, ...) %27(ptr %26, ptr @19, i64 100, i64 200)
// ESCAPE-NEXT:   ret void
// ESCAPE-EMPTY:
// ESCAPE-NEXT: _llgo_3:                                          ; preds = %_llgo_0
// ESCAPE-NEXT:   %29 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %7, 1
// ESCAPE-NEXT:   %30 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface$a85zs5wWQQoPIERm_en8plssh4spdIeeXZPC-E0TDh0", ptr %8)
// ESCAPE-NEXT:   %31 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %30, 0
// ESCAPE-NEXT:   %32 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %31, ptr %29, 1
// ESCAPE-NEXT:   %33 = insertvalue { %"{{.*}}/runtime/internal/runtime.iface", i1 } undef, %"{{.*}}/runtime/internal/runtime.iface" %32, 0
// ESCAPE-NEXT:   %34 = insertvalue { %"{{.*}}/runtime/internal/runtime.iface", i1 } %33, i1 true, 1
// ESCAPE-NEXT:   br label %_llgo_5
// ESCAPE-EMPTY:
// ESCAPE-NEXT: _llgo_4:                                          ; preds = %_llgo_0
// ESCAPE-NEXT:   br label %_llgo_5
// ESCAPE-EMPTY:
// ESCAPE-NEXT: _llgo_5:                                          ; preds = %_llgo_4, %_llgo_3
// ESCAPE-NEXT:   %35 = phi { %"{{.*}}/runtime/internal/runtime.iface", i1 } [ %34, %_llgo_3 ], [ zeroinitializer, %_llgo_4 ]
// ESCAPE-NEXT:   %36 = extractvalue { %"{{.*}}/runtime/internal/runtime.iface", i1 } %35, 0
// ESCAPE-NEXT:   %37 = extractvalue { %"{{.*}}/runtime/internal/runtime.iface", i1 } %35, 1
// ESCAPE-NEXT:   br i1 %37, label %_llgo_2, label %_llgo_1
// ESCAPE-NEXT: }
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
