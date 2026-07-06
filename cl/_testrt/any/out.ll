; ModuleID = 'github.com/goplus/llgo/cl/_testrt/any'
source_filename = "github.com/goplus/llgo/cl/_testrt/any"
target datalayout = "e-m:o-i64:64-i128:128-n32:64-S128-Fn32"
target triple = "arm64-apple-macosx"

%"github.com/goplus/llgo/runtime/abi.PtrType" = type { %"github.com/goplus/llgo/runtime/abi.Type", ptr }
%"github.com/goplus/llgo/runtime/abi.Type" = type { i64, i64, i32, i8, i8, i8, i8, { ptr, ptr }, ptr, %"github.com/goplus/llgo/runtime/internal/runtime.String", ptr }
%"github.com/goplus/llgo/runtime/internal/runtime.String" = type { ptr, i64 }
%"github.com/goplus/llgo/runtime/abi.InterfaceType" = type { %"github.com/goplus/llgo/runtime/abi.Type", %"github.com/goplus/llgo/runtime/internal/runtime.String", %"github.com/goplus/llgo/runtime/internal/runtime.Slice" }
%"github.com/goplus/llgo/runtime/internal/runtime.Slice" = type { ptr, i64, i64 }
%"github.com/goplus/llgo/runtime/internal/runtime.eface" = type { ptr, ptr }

@"github.com/goplus/llgo/cl/_testrt/any.init$guard" = global i1 false, align 1
@"*_llgo_int8" = weak_odr constant %"github.com/goplus/llgo/runtime/abi.PtrType" { %"github.com/goplus/llgo/runtime/abi.Type" { i64 8, i64 8, i32 -1399554408, i8 10, i8 8, i8 8, i8 54, { ptr, ptr } { ptr @"__llgo_stub.github.com/goplus/llgo/runtime/internal/runtime.memequalptr", ptr null }, ptr null, %"github.com/goplus/llgo/runtime/internal/runtime.String" { ptr @0, i64 4 }, ptr null }, ptr @_llgo_int8 }, align 8
@0 = private unnamed_addr constant [4 x i8] c"int8", align 1
@_llgo_int8 = weak_odr constant %"github.com/goplus/llgo/runtime/abi.Type" { i64 1, i64 0, i32 1444672578, i8 12, i8 1, i8 1, i8 3, { ptr, ptr } { ptr @"__llgo_stub.github.com/goplus/llgo/runtime/internal/runtime.memequal8", ptr null }, ptr null, %"github.com/goplus/llgo/runtime/internal/runtime.String" { ptr @0, i64 4 }, ptr @"*_llgo_int8" }, align 8
@_llgo_any = weak_odr constant %"github.com/goplus/llgo/runtime/abi.InterfaceType" { %"github.com/goplus/llgo/runtime/abi.Type" { i64 16, i64 16, i32 1376530322, i8 0, i8 8, i8 8, i8 20, { ptr, ptr } { ptr @"__llgo_stub.github.com/goplus/llgo/runtime/internal/runtime.nilinterequal", ptr null }, ptr null, %"github.com/goplus/llgo/runtime/internal/runtime.String" { ptr @1, i64 12 }, ptr @"*_llgo_any" }, %"github.com/goplus/llgo/runtime/internal/runtime.String" { ptr @2, i64 37 }, %"github.com/goplus/llgo/runtime/internal/runtime.Slice" zeroinitializer }, align 8
@1 = private unnamed_addr constant [12 x i8] c"interface {}", align 1
@"*_llgo_any" = weak_odr constant %"github.com/goplus/llgo/runtime/abi.PtrType" { %"github.com/goplus/llgo/runtime/abi.Type" { i64 8, i64 8, i32 1741196194, i8 10, i8 8, i8 8, i8 54, { ptr, ptr } { ptr @"__llgo_stub.github.com/goplus/llgo/runtime/internal/runtime.memequalptr", ptr null }, ptr null, %"github.com/goplus/llgo/runtime/internal/runtime.String" { ptr @1, i64 12 }, ptr null }, ptr @_llgo_any }, align 8
@2 = private unnamed_addr constant [37 x i8] c"github.com/goplus/llgo/cl/_testrt/any", align 1
@_llgo_int = weak_odr constant %"github.com/goplus/llgo/runtime/abi.Type" { i64 8, i64 0, i32 -25294021, i8 12, i8 8, i8 8, i8 2, { ptr, ptr } { ptr @"__llgo_stub.github.com/goplus/llgo/runtime/internal/runtime.memequal64", ptr null }, ptr null, %"github.com/goplus/llgo/runtime/internal/runtime.String" { ptr @3, i64 3 }, ptr @"*_llgo_int" }, align 8
@3 = private unnamed_addr constant [3 x i8] c"int", align 1
@"*_llgo_int" = weak_odr constant %"github.com/goplus/llgo/runtime/abi.PtrType" { %"github.com/goplus/llgo/runtime/abi.Type" { i64 8, i64 8, i32 -939606833, i8 10, i8 8, i8 8, i8 54, { ptr, ptr } { ptr @"__llgo_stub.github.com/goplus/llgo/runtime/internal/runtime.memequalptr", ptr null }, ptr null, %"github.com/goplus/llgo/runtime/internal/runtime.String" { ptr @3, i64 3 }, ptr null }, ptr @_llgo_int }, align 8
@4 = private unnamed_addr constant [7 x i8] c"%s %d\0A\00", align 1
@5 = private unnamed_addr constant [6 x i8] c"Hello\00", align 1

; Function Attrs: null_pointer_is_valid
define ptr @"github.com/goplus/llgo/cl/_testrt/any.hi"(%"github.com/goplus/llgo/runtime/internal/runtime.eface" %0) #0 {
_llgo_0:
  %1 = extractvalue %"github.com/goplus/llgo/runtime/internal/runtime.eface" %0, 0
  %2 = icmp eq ptr %1, @"*_llgo_int8"
  br i1 %2, label %_llgo_1, label %_llgo_2

_llgo_1:                                          ; preds = %_llgo_0
  %3 = extractvalue %"github.com/goplus/llgo/runtime/internal/runtime.eface" %0, 1
  ret ptr %3

_llgo_2:                                          ; preds = %_llgo_0
  %4 = call %"github.com/goplus/llgo/runtime/internal/runtime.eface" @"github.com/goplus/llgo/runtime/internal/runtime.TypeAssertError"(ptr %1, ptr @"*_llgo_int8", ptr @_llgo_any)
  call void @"github.com/goplus/llgo/runtime/internal/runtime.Panic"(%"github.com/goplus/llgo/runtime/internal/runtime.eface" %4)
  unreachable
}

; Function Attrs: null_pointer_is_valid
define i64 @"github.com/goplus/llgo/cl/_testrt/any.incVal"(%"github.com/goplus/llgo/runtime/internal/runtime.eface" %0) #0 {
_llgo_0:
  %1 = extractvalue %"github.com/goplus/llgo/runtime/internal/runtime.eface" %0, 0
  %2 = icmp eq ptr %1, @_llgo_int
  br i1 %2, label %_llgo_1, label %_llgo_2

_llgo_1:                                          ; preds = %_llgo_0
  %3 = extractvalue %"github.com/goplus/llgo/runtime/internal/runtime.eface" %0, 1
  %4 = load i64, ptr %3, align 8
  %5 = add i64 %4, 1
  ret i64 %5

_llgo_2:                                          ; preds = %_llgo_0
  %6 = call %"github.com/goplus/llgo/runtime/internal/runtime.eface" @"github.com/goplus/llgo/runtime/internal/runtime.TypeAssertError"(ptr %1, ptr @_llgo_int, ptr @_llgo_any)
  call void @"github.com/goplus/llgo/runtime/internal/runtime.Panic"(%"github.com/goplus/llgo/runtime/internal/runtime.eface" %6)
  unreachable
}

; Function Attrs: null_pointer_is_valid
define void @"github.com/goplus/llgo/cl/_testrt/any.init"() #0 {
_llgo_0:
  %0 = load i1, ptr @"github.com/goplus/llgo/cl/_testrt/any.init$guard", align 1
  br i1 %0, label %_llgo_2, label %_llgo_1

_llgo_1:                                          ; preds = %_llgo_0
  store i1 true, ptr @"github.com/goplus/llgo/cl/_testrt/any.init$guard", align 1
  br label %_llgo_2

_llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
  ret void
}

; Function Attrs: null_pointer_is_valid
define void @"github.com/goplus/llgo/cl/_testrt/any.main"() #0 {
_llgo_0:
  %0 = call ptr @"github.com/goplus/llgo/cl/_testrt/any.hi"(%"github.com/goplus/llgo/runtime/internal/runtime.eface" { ptr @"*_llgo_int8", ptr @5 })
  %1 = call ptr @"github.com/goplus/llgo/runtime/internal/runtime.AllocU"(i64 8)
  store i64 100, ptr %1, align 8
  %2 = insertvalue %"github.com/goplus/llgo/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %1, 1
  %3 = call i64 @"github.com/goplus/llgo/cl/_testrt/any.incVal"(%"github.com/goplus/llgo/runtime/internal/runtime.eface" %2)
  %4 = call i32 (ptr, ...) @printf(ptr @4, ptr %0, i64 %3)
  ret void
}

; Function Attrs: null_pointer_is_valid
declare i1 @"github.com/goplus/llgo/runtime/internal/runtime.memequalptr"(ptr, ptr) #0

define linkonce i1 @"__llgo_stub.github.com/goplus/llgo/runtime/internal/runtime.memequalptr"(ptr %0, ptr %1, ptr %2) {
_llgo_0:
  %3 = tail call i1 @"github.com/goplus/llgo/runtime/internal/runtime.memequalptr"(ptr %1, ptr %2)
  ret i1 %3
}

; Function Attrs: null_pointer_is_valid
declare i1 @"github.com/goplus/llgo/runtime/internal/runtime.memequal8"(ptr, ptr) #0

define linkonce i1 @"__llgo_stub.github.com/goplus/llgo/runtime/internal/runtime.memequal8"(ptr %0, ptr %1, ptr %2) {
_llgo_0:
  %3 = tail call i1 @"github.com/goplus/llgo/runtime/internal/runtime.memequal8"(ptr %1, ptr %2)
  ret i1 %3
}

; Function Attrs: null_pointer_is_valid
declare %"github.com/goplus/llgo/runtime/internal/runtime.eface" @"github.com/goplus/llgo/runtime/internal/runtime.TypeAssertError"(ptr, ptr, ptr) #0

; Function Attrs: null_pointer_is_valid
declare i1 @"github.com/goplus/llgo/runtime/internal/runtime.nilinterequal"(ptr, ptr) #0

define linkonce i1 @"__llgo_stub.github.com/goplus/llgo/runtime/internal/runtime.nilinterequal"(ptr %0, ptr %1, ptr %2) {
_llgo_0:
  %3 = tail call i1 @"github.com/goplus/llgo/runtime/internal/runtime.nilinterequal"(ptr %1, ptr %2)
  ret i1 %3
}

; Function Attrs: null_pointer_is_valid
declare void @"github.com/goplus/llgo/runtime/internal/runtime.Panic"(%"github.com/goplus/llgo/runtime/internal/runtime.eface") #0

; Function Attrs: null_pointer_is_valid
declare i1 @"github.com/goplus/llgo/runtime/internal/runtime.memequal64"(ptr, ptr) #0

define linkonce i1 @"__llgo_stub.github.com/goplus/llgo/runtime/internal/runtime.memequal64"(ptr %0, ptr %1, ptr %2) {
_llgo_0:
  %3 = tail call i1 @"github.com/goplus/llgo/runtime/internal/runtime.memequal64"(ptr %1, ptr %2)
  ret i1 %3
}

; Function Attrs: null_pointer_is_valid
declare ptr @"github.com/goplus/llgo/runtime/internal/runtime.AllocU"(i64) #0

declare i32 @printf(ptr, ...)

attributes #0 = { null_pointer_is_valid "frame-pointer"="non-leaf" }
