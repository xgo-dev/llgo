// LITTEST
package main

import (
	"unsafe"

	"github.com/goplus/llgo/runtime/abi"
)

// CHECK-LINE: @0 = private unnamed_addr constant [6 x i8] c"invoke", align 1
// CHECK-LINE: @1 = private unnamed_addr constant [7 x i8] c"\09elem: ", align 1
// CHECK-LINE: @2 = private unnamed_addr constant [9 x i8] c"\09uncomm: ", align 1
// CHECK-LINE: @23 = private unnamed_addr constant [5 x i8] c"hello", align 1

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/eface.(*T).Invoke"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 6 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

func (t *T) Invoke() {
	println("invoke")
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/eface.dump"(%"{{.*}}/runtime/internal/runtime.eface" %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %0, ptr %1, align 8
// CHECK-NEXT:   %2 = icmp eq ptr %1, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %2)
// CHECK-NEXT:   %3 = getelementptr inbounds %"{{.*}}/cl/_testrt/eface.eface", ptr %1, i32 0, i32 0
// CHECK-NEXT:   %4 = load ptr, ptr %3, align 8
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/eface.dumpTyp"(ptr %4, %"{{.*}}/runtime/internal/runtime.String" zeroinitializer)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

func dump(v any) {
	e := (*eface)(unsafe.Pointer(&v))
	dumpTyp(e._type, "")
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/eface.dumpTyp"(ptr %0, %"{{.*}}/runtime/internal/runtime.String" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %1)
// CHECK-NEXT:   %2 = call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/runtime/abi.(*Type).String"(ptr %0)
// CHECK-NEXT:   %3 = call i64 @"{{.*}}/runtime/abi.(*Type).Kind"(ptr %0)
// CHECK-NEXT:   %4 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %4)
// CHECK-NEXT:   %5 = getelementptr inbounds %"{{.*}}/runtime/abi.Type", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %6 = load i64, ptr %5, align 8
// CHECK-NEXT:   %7 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %7)
// CHECK-NEXT:   %8 = getelementptr inbounds %"{{.*}}/runtime/abi.Type", ptr %0, i32 0, i32 1
// CHECK-NEXT:   %9 = load i64, ptr %8, align 8
// CHECK-NEXT:   %10 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %10)
// CHECK-NEXT:   %11 = getelementptr inbounds %"{{.*}}/runtime/abi.Type", ptr %0, i32 0, i32 2
// CHECK-NEXT:   %12 = load i32, ptr %11, align 4
// CHECK-NEXT:   %13 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %13)
// CHECK-NEXT:   %14 = getelementptr inbounds %"{{.*}}/runtime/abi.Type", ptr %0, i32 0, i32 3
// CHECK-NEXT:   %15 = load i8, ptr %14, align 1
// CHECK-NEXT:   %16 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %16)
// CHECK-NEXT:   %17 = getelementptr inbounds %"{{.*}}/runtime/abi.Type", ptr %0, i32 0, i32 4
// CHECK-NEXT:   %18 = load i8, ptr %17, align 1
// CHECK-NEXT:   %19 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %19)
// CHECK-NEXT:   %20 = getelementptr inbounds %"{{.*}}/runtime/abi.Type", ptr %0, i32 0, i32 10
// CHECK-NEXT:   %21 = load ptr, ptr %20, align 8
// CHECK-NEXT:   %22 = call ptr @"{{.*}}/runtime/abi.(*Type).Uncommon"(ptr %0)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %2)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintUint"(i64 %3)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintUint"(i64 %6)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintUint"(i64 %9)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   %23 = zext i32 %12 to i64
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintUint"(i64 %23)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   %24 = zext i8 %15 to i64
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintUint"(i64 %24)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   %25 = zext i8 %18 to i64
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintUint"(i64 %25)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintPointer"(ptr %21)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintPointer"(ptr %22)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %26 = call ptr @"{{.*}}/runtime/abi.(*Type).Elem"(ptr %0)
// CHECK-NEXT:   %27 = icmp ne ptr %26, null
// CHECK-NEXT:   br i1 %27, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %28 = call ptr @"{{.*}}/runtime/abi.(*Type).Elem"(ptr %0)
// CHECK-NEXT:   %29 = call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/runtime/internal/runtime.StringCat"(%"{{.*}}/runtime/internal/runtime.String" %1, %"{{.*}}/runtime/internal/runtime.String" { ptr @1, i64 7 })
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/eface.dumpTyp"(ptr %28, %"{{.*}}/runtime/internal/runtime.String" %29)
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   %30 = call ptr @"{{.*}}/runtime/abi.(*Type).Uncommon"(ptr %0)
// CHECK-NEXT:   %31 = icmp ne ptr %30, null
// CHECK-NEXT:   br i1 %31, label %_llgo_3, label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %32 = call ptr @"{{.*}}/runtime/abi.(*Type).Uncommon"(ptr %0)
// CHECK-NEXT:   %33 = call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/runtime/internal/runtime.StringCat"(%"{{.*}}/runtime/internal/runtime.String" %1, %"{{.*}}/runtime/internal/runtime.String" { ptr @2, i64 9 })
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/eface.dumpUncommon"(ptr %32, %"{{.*}}/runtime/internal/runtime.String" %33)
// CHECK-NEXT:   %34 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %34)
// CHECK-NEXT:   %35 = getelementptr inbounds %"{{.*}}/runtime/abi.Type", ptr %0, i32 0, i32 10
// CHECK-NEXT:   %36 = load ptr, ptr %35, align 8
// CHECK-NEXT:   %37 = icmp ne ptr %36, null
// CHECK-NEXT:   br i1 %37, label %_llgo_5, label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_5, %_llgo_3, %_llgo_2
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_3
// CHECK-NEXT:   %38 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %38)
// CHECK-NEXT:   %39 = getelementptr inbounds %"{{.*}}/runtime/abi.Type", ptr %0, i32 0, i32 10
// CHECK-NEXT:   %40 = load ptr, ptr %39, align 8
// CHECK-NEXT:   %41 = call ptr @"{{.*}}/runtime/abi.(*Type).Uncommon"(ptr %40)
// CHECK-NEXT:   %42 = call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/runtime/internal/runtime.StringCat"(%"{{.*}}/runtime/internal/runtime.String" %1, %"{{.*}}/runtime/internal/runtime.String" { ptr @2, i64 9 })
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/eface.dumpUncommon"(ptr %41, %"{{.*}}/runtime/internal/runtime.String" %42)
// CHECK-NEXT:   br label %_llgo_4
// CHECK-NEXT: }

func dumpTyp(t *abi.Type, sep string) {
	print(sep)
	println(t.String(), t.Kind(), t.Size_, t.PtrBytes, t.Hash, t.TFlag, t.Align_, t.PtrToThis_, t.Uncommon())
	if t.Elem() != nil {
		dumpTyp(t.Elem(), sep+"\telem: ")
	}
	if t.Uncommon() != nil {
		dumpUncommon(t.Uncommon(), sep+"\tuncomm: ")
		if t.PtrToThis_ != nil {
			dumpUncommon(t.PtrToThis_.Uncommon(), sep+"\tuncomm: ")
		}
	}
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/eface.dumpUncommon"(ptr %0, %"{{.*}}/runtime/internal/runtime.String" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %1)
// CHECK-NEXT:   %2 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %2)
// CHECK-NEXT:   %3 = getelementptr inbounds %"{{.*}}/runtime/abi.UncommonType", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %4 = load %"{{.*}}/runtime/internal/runtime.String", ptr %3, align 8
// CHECK-NEXT:   %5 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %5)
// CHECK-NEXT:   %6 = getelementptr inbounds %"{{.*}}/runtime/abi.UncommonType", ptr %0, i32 0, i32 1
// CHECK-NEXT:   %7 = load i16, ptr %6, align 2
// CHECK-NEXT:   %8 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %8)
// CHECK-NEXT:   %9 = getelementptr inbounds %"{{.*}}/runtime/abi.UncommonType", ptr %0, i32 0, i32 2
// CHECK-NEXT:   %10 = load i16, ptr %9, align 2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %4)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   %11 = zext i16 %7 to i64
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintUint"(i64 %11)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   %12 = zext i16 %10 to i64
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintUint"(i64 %12)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

func dumpUncommon(u *abi.UncommonType, sep string) {
	print(sep)
	println(u.PkgPath_, u.Mcount, u.Xcount)
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/eface.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testrt/eface.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testrt/eface.init$guard", align 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/abi.init"()
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

type T string

type eface struct {
	_type *abi.Type
	data  unsafe.Pointer
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/eface.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// CHECK-NEXT:   store i1 true, ptr %0, align 1
// CHECK-NEXT:   %1 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_bool, ptr undef }, ptr %0, 1
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/eface.dump"(%"{{.*}}/runtime/internal/runtime.eface" %1)
// CHECK-NEXT:   %2 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 0, ptr %2, align 8
// CHECK-NEXT:   %3 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %2, 1
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/eface.dump"(%"{{.*}}/runtime/internal/runtime.eface" %3)
// CHECK-NEXT:   %4 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// CHECK-NEXT:   store i8 0, ptr %4, align 1
// CHECK-NEXT:   %5 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int8, ptr undef }, ptr %4, 1
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/eface.dump"(%"{{.*}}/runtime/internal/runtime.eface" %5)
// CHECK-NEXT:   %6 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 2)
// CHECK-NEXT:   store i16 0, ptr %6, align 2
// CHECK-NEXT:   %7 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int16, ptr undef }, ptr %6, 1
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/eface.dump"(%"{{.*}}/runtime/internal/runtime.eface" %7)
// CHECK-NEXT:   %8 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store i32 0, ptr %8, align 4
// CHECK-NEXT:   %9 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int32, ptr undef }, ptr %8, 1
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/eface.dump"(%"{{.*}}/runtime/internal/runtime.eface" %9)
// CHECK-NEXT:   %10 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 0, ptr %10, align 8
// CHECK-NEXT:   %11 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int64, ptr undef }, ptr %10, 1
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/eface.dump"(%"{{.*}}/runtime/internal/runtime.eface" %11)
// CHECK-NEXT:   %12 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 0, ptr %12, align 8
// CHECK-NEXT:   %13 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint, ptr undef }, ptr %12, 1
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/eface.dump"(%"{{.*}}/runtime/internal/runtime.eface" %13)
// CHECK-NEXT:   %14 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// CHECK-NEXT:   store i8 0, ptr %14, align 1
// CHECK-NEXT:   %15 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint8, ptr undef }, ptr %14, 1
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/eface.dump"(%"{{.*}}/runtime/internal/runtime.eface" %15)
// CHECK-NEXT:   %16 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 2)
// CHECK-NEXT:   store i16 0, ptr %16, align 2
// CHECK-NEXT:   %17 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint16, ptr undef }, ptr %16, 1
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/eface.dump"(%"{{.*}}/runtime/internal/runtime.eface" %17)
// CHECK-NEXT:   %18 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store i32 0, ptr %18, align 4
// CHECK-NEXT:   %19 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint32, ptr undef }, ptr %18, 1
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/eface.dump"(%"{{.*}}/runtime/internal/runtime.eface" %19)
// CHECK-NEXT:   %20 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 0, ptr %20, align 8
// CHECK-NEXT:   %21 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint64, ptr undef }, ptr %20, 1
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/eface.dump"(%"{{.*}}/runtime/internal/runtime.eface" %21)
// CHECK-NEXT:   %22 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 0, ptr %22, align 8
// CHECK-NEXT:   %23 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uintptr, ptr undef }, ptr %22, 1
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/eface.dump"(%"{{.*}}/runtime/internal/runtime.eface" %23)
// CHECK-NEXT:   %24 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store float 0.000000e+00, ptr %24, align 4
// CHECK-NEXT:   %25 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_float32, ptr undef }, ptr %24, 1
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/eface.dump"(%"{{.*}}/runtime/internal/runtime.eface" %25)
// CHECK-NEXT:   %26 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store double 0.000000e+00, ptr %26, align 8
// CHECK-NEXT:   %27 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_float64, ptr undef }, ptr %26, 1
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/eface.dump"(%"{{.*}}/runtime/internal/runtime.eface" %27)
// CHECK-NEXT:   %28 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 80)
// CHECK-NEXT:   store [10 x i64] zeroinitializer, ptr %28, align 8
// CHECK-NEXT:   %29 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"[10]_llgo_int", ptr undef }, ptr %28, 1
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/eface.dump"(%"{{.*}}/runtime/internal/runtime.eface" %29)
// CHECK-NEXT:   %30 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/cl/_testrt/eface.main$1", ptr null }, ptr %30, align 8
// CHECK-NEXT:   %31 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"_llgo_closure$b7Su1hWaFih-M0M9hMk6nO_RD1K_GQu5WjIXQp6Q2e8", ptr undef }, ptr %30, 1
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/eface.dump"(%"{{.*}}/runtime/internal/runtime.eface" %31)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/eface.dump"(%"{{.*}}/runtime/internal/runtime.eface" { ptr @"*_llgo_int", ptr null })
// CHECK-NEXT:   %32 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" { ptr @"__llgo.moduleZeroSizedAlloc$", i64 0, i64 0 }, ptr %32, align 8
// CHECK-NEXT:   %33 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"[]_llgo_int", ptr undef }, ptr %32, 1
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/eface.dump"(%"{{.*}}/runtime/internal/runtime.eface" %33)
// CHECK-NEXT:   %34 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @23, i64 5 }, ptr %34, align 8
// CHECK-NEXT:   %35 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %34, 1
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/eface.dump"(%"{{.*}}/runtime/internal/runtime.eface" %35)
// CHECK-NEXT:   %36 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store { i8, i64, i64 } zeroinitializer, ptr %36, align 8
// CHECK-NEXT:   %37 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"{{.*}}/cl/_testrt/eface.struct$RKbUG45GE4henGMAdmt0Rju0JptyR8NsX7IZLsOI0OM", ptr undef }, ptr %36, 1
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/eface.dump"(%"{{.*}}/runtime/internal/runtime.eface" %37)
// CHECK-NEXT:   %38 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" zeroinitializer, ptr %38, align 8
// CHECK-NEXT:   %39 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"_llgo_{{.*}}/cl/_testrt/eface.T", ptr undef }, ptr %38, 1
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/eface.dump"(%"{{.*}}/runtime/internal/runtime.eface" %39)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

func main() {
	dump(true)
	dump(0)
	dump(int8(0))
	dump(int16(0))
	dump(int32(0))
	dump(int64(0))
	dump(uint(0))
	dump(uint8(0))
	dump(uint16(0))
	dump(uint32(0))
	dump(uint64(0))
	dump(uintptr(0))
	dump(float32(0))
	dump(float64(0))
	dump([10]int{})

	// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/eface.main$1"(){{.*}} {
	// CHECK-NEXT: _llgo_0:
	// CHECK-NEXT:   ret void
	// CHECK-NEXT: }

	dump(func() {})
	dump((*int)(nil))
	dump([]int{})
	dump("hello")
	dump(struct {
		x int8
		y int
		z int
	}{})
	var t T
	dump(t)
}

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequal8"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.memequal8"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequal64"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.memequal64"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequal16"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.memequal16"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }

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

// CHECK-LABEL: define linkonce void @"__llgo_stub.{{.*}}/cl/_testrt/eface.main$1"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   tail call void @"{{.*}}/cl/_testrt/eface.main$1"()
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce void @"__llgo_stub.{{.*}}/cl/_testrt/eface.(*T).Invoke"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   tail call void @"{{.*}}/cl/_testrt/eface.(*T).Invoke"(ptr %1)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
