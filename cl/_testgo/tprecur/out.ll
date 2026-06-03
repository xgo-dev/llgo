; ModuleID = 'github.com/goplus/llgo/cl/_testgo/tprecur'
source_filename = "github.com/goplus/llgo/cl/_testgo/tprecur"

%"github.com/goplus/llgo/runtime/abi.Type" = type { i64, i64, i32, i8, i8, i8, i8, { ptr, ptr }, ptr, %"github.com/goplus/llgo/runtime/internal/runtime.String", ptr }
%"github.com/goplus/llgo/runtime/internal/runtime.String" = type { ptr, i64 }
%"github.com/goplus/llgo/runtime/abi.PtrType" = type { %"github.com/goplus/llgo/runtime/abi.Type", ptr }
%"github.com/goplus/llgo/runtime/internal/runtime.eface" = type { ptr, ptr }
%"github.com/goplus/llgo/runtime/internal/runtime.Slice" = type { ptr, i64, i64 }

@"github.com/goplus/llgo/cl/_testgo/tprecur.init$guard" = global i1 false, align 1
@0 = private unnamed_addr constant [5 x i8] c"error", align 1
@_llgo_string = weak_odr constant %"github.com/goplus/llgo/runtime/abi.Type" { i64 16, i64 8, i32 1749264893, i8 4, i8 8, i8 8, i8 24, { ptr, ptr } { ptr @"__llgo_stub.github.com/goplus/llgo/runtime/internal/runtime.strequal", ptr null }, ptr null, %"github.com/goplus/llgo/runtime/internal/runtime.String" { ptr @1, i64 6 }, ptr @"*_llgo_string" }, align 8
@1 = private unnamed_addr constant [6 x i8] c"string", align 1
@"*_llgo_string" = weak_odr constant %"github.com/goplus/llgo/runtime/abi.PtrType" { %"github.com/goplus/llgo/runtime/abi.Type" { i64 8, i64 8, i32 -1323879264, i8 10, i8 8, i8 8, i8 54, { ptr, ptr } { ptr @"__llgo_stub.github.com/goplus/llgo/runtime/internal/runtime.memequalptr", ptr null }, ptr null, %"github.com/goplus/llgo/runtime/internal/runtime.String" { ptr @1, i64 6 }, ptr null }, ptr @_llgo_string }, align 8

define void @"github.com/goplus/llgo/cl/_testgo/tprecur.init"() {
_llgo_0:
  %0 = load i1, ptr @"github.com/goplus/llgo/cl/_testgo/tprecur.init$guard", align 1
  br i1 %0, label %_llgo_2, label %_llgo_1

_llgo_1:                                          ; preds = %_llgo_0
  store i1 true, ptr @"github.com/goplus/llgo/cl/_testgo/tprecur.init$guard", align 1
  br label %_llgo_2

_llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
  ret void
}

define void @"github.com/goplus/llgo/cl/_testgo/tprecur.main"() {
_llgo_0:
  call void @"github.com/goplus/llgo/cl/_testgo/tprecur.recursive"()
  ret void
}

define void @"github.com/goplus/llgo/cl/_testgo/tprecur.recursive"() {
_llgo_0:
  %0 = call i64 @"github.com/goplus/llgo/cl/_testgo/tprecur.recur1[github.com/goplus/llgo/cl/_testgo/tprecur.T.1.0]"(i64 5)
  %1 = icmp ne i64 %0, 110
  br i1 %1, label %_llgo_1, label %_llgo_2

_llgo_1:                                          ; preds = %_llgo_0
  %2 = call ptr @"github.com/goplus/llgo/runtime/internal/runtime.AllocU"(i64 16)
  store %"github.com/goplus/llgo/runtime/internal/runtime.String" { ptr @0, i64 5 }, ptr %2, align 8
  %3 = insertvalue %"github.com/goplus/llgo/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %2, 1
  call void @"github.com/goplus/llgo/runtime/internal/runtime.Panic"(%"github.com/goplus/llgo/runtime/internal/runtime.eface" %3)
  unreachable

_llgo_2:                                          ; preds = %_llgo_0
  ret void
}

define linkonce i64 @"github.com/goplus/llgo/cl/_testgo/tprecur.recur1[github.com/goplus/llgo/cl/_testgo/tprecur.T.1.0]"(i64 %0) {
_llgo_0:
  %1 = icmp eq i64 %0, 0
  br i1 %1, label %_llgo_1, label %_llgo_3

_llgo_1:                                          ; preds = %_llgo_3, %_llgo_0
  ret i64 1

_llgo_2:                                          ; preds = %_llgo_3
  %2 = sub i64 %0, 1
  %3 = call i64 @"github.com/goplus/llgo/cl/_testgo/tprecur.recur2[github.com/goplus/llgo/cl/_testgo/tprecur.T.1.0]"(i64 %2)
  %4 = mul i64 %0, %3
  ret i64 %4

_llgo_3:                                          ; preds = %_llgo_0
  %5 = icmp eq i64 %0, 1
  br i1 %5, label %_llgo_1, label %_llgo_2
}

declare i1 @"github.com/goplus/llgo/runtime/internal/runtime.strequal"(ptr, ptr)

define linkonce i1 @"__llgo_stub.github.com/goplus/llgo/runtime/internal/runtime.strequal"(ptr %0, ptr %1, ptr %2) {
_llgo_0:
  %3 = tail call i1 @"github.com/goplus/llgo/runtime/internal/runtime.strequal"(ptr %1, ptr %2)
  ret i1 %3
}

declare i1 @"github.com/goplus/llgo/runtime/internal/runtime.memequalptr"(ptr, ptr)

define linkonce i1 @"__llgo_stub.github.com/goplus/llgo/runtime/internal/runtime.memequalptr"(ptr %0, ptr %1, ptr %2) {
_llgo_0:
  %3 = tail call i1 @"github.com/goplus/llgo/runtime/internal/runtime.memequalptr"(ptr %1, ptr %2)
  ret i1 %3
}

declare ptr @"github.com/goplus/llgo/runtime/internal/runtime.AllocU"(i64)

declare void @"github.com/goplus/llgo/runtime/internal/runtime.Panic"(%"github.com/goplus/llgo/runtime/internal/runtime.eface")

define linkonce i64 @"github.com/goplus/llgo/cl/_testgo/tprecur.recur2[github.com/goplus/llgo/cl/_testgo/tprecur.T.1.0]"(i64 %0) {
_llgo_0:
  %1 = call %"github.com/goplus/llgo/runtime/internal/runtime.Slice" @"github.com/goplus/llgo/runtime/internal/runtime.MakeSlice"(i64 %0, i64 %0, i64 8)
  %2 = extractvalue %"github.com/goplus/llgo/runtime/internal/runtime.Slice" %1, 1
  br label %_llgo_1

_llgo_1:                                          ; preds = %_llgo_2, %_llgo_0
  %3 = phi i64 [ -1, %_llgo_0 ], [ %4, %_llgo_2 ]
  %4 = add i64 %3, 1
  %5 = icmp slt i64 %4, %2
  br i1 %5, label %_llgo_2, label %_llgo_3

_llgo_2:                                          ; preds = %_llgo_1
  %6 = add i64 %4, 1
  %7 = extractvalue %"github.com/goplus/llgo/runtime/internal/runtime.Slice" %1, 0
  %8 = extractvalue %"github.com/goplus/llgo/runtime/internal/runtime.Slice" %1, 1
  %9 = icmp slt i64 %4, 0
  %10 = icmp uge i64 %4, %8
  %11 = or i1 %10, %9
  call void @"github.com/goplus/llgo/runtime/internal/runtime.AssertIndexRange"(i1 %11)
  %12 = getelementptr inbounds i64, ptr %7, i64 %4
  store i64 %6, ptr %12, align 4
  br label %_llgo_1

_llgo_3:                                          ; preds = %_llgo_1
  %13 = extractvalue %"github.com/goplus/llgo/runtime/internal/runtime.Slice" %1, 1
  br label %_llgo_4

_llgo_4:                                          ; preds = %_llgo_5, %_llgo_3
  %14 = phi i64 [ 0, %_llgo_3 ], [ %25, %_llgo_5 ]
  %15 = phi i64 [ -1, %_llgo_3 ], [ %16, %_llgo_5 ]
  %16 = add i64 %15, 1
  %17 = icmp slt i64 %16, %13
  br i1 %17, label %_llgo_5, label %_llgo_6

_llgo_5:                                          ; preds = %_llgo_4
  %18 = extractvalue %"github.com/goplus/llgo/runtime/internal/runtime.Slice" %1, 0
  %19 = extractvalue %"github.com/goplus/llgo/runtime/internal/runtime.Slice" %1, 1
  %20 = icmp slt i64 %16, 0
  %21 = icmp uge i64 %16, %19
  %22 = or i1 %21, %20
  call void @"github.com/goplus/llgo/runtime/internal/runtime.AssertIndexRange"(i1 %22)
  %23 = getelementptr inbounds i64, ptr %18, i64 %16
  %24 = load i64, ptr %23, align 4
  %25 = add i64 %14, %24
  br label %_llgo_4

_llgo_6:                                          ; preds = %_llgo_4
  %26 = sub i64 %0, 1
  %27 = call i64 @"github.com/goplus/llgo/cl/_testgo/tprecur.recur1[github.com/goplus/llgo/cl/_testgo/tprecur.T.1.0]"(i64 %26)
  %28 = add i64 %14, %27
  ret i64 %28
}

declare %"github.com/goplus/llgo/runtime/internal/runtime.Slice" @"github.com/goplus/llgo/runtime/internal/runtime.MakeSlice"(i64, i64, i64)

declare void @"github.com/goplus/llgo/runtime/internal/runtime.AssertIndexRange"(i1)
