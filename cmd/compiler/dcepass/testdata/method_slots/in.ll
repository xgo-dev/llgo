%"runtime.String" = type { ptr, i64 }
%"runtime.Slice" = type { ptr, i64, i64 }
%"runtime/abi.Type" = type { i64, i64, i32, i8, i8, i8, i8, { ptr, ptr }, ptr, %"runtime.String", ptr }
%"runtime/abi.StructType" = type { %"runtime/abi.Type", %"runtime.String", %"runtime.Slice" }
%"runtime/abi.PtrType" = type { %"runtime/abi.Type", ptr }
%"runtime/abi.UncommonType" = type { %"runtime.String", i16, i16, i32 }
%"github.com/xgo-dev/llgo/runtime/abi.Method" = type { %"runtime.String", ptr, ptr, ptr }
%"runtime/abi.FuncType" = type { %"runtime/abi.Type", %"runtime.Slice", %"runtime.Slice" }
%"runtime/abi.InterfaceType" = type { %"runtime/abi.Type", %"runtime.String", %"runtime.Slice" }
%"runtime/abi.Imethod" = type { %"runtime.String", ptr }
%Task = type {}

@task.name = private unnamed_addr constant [9 x i8] c"main.Task", align 1
@task.pkg = private unnamed_addr constant [4 x i8] c"main", align 1
@method.drop = private unnamed_addr constant [4 x i8] c"Drop", align 1
@method.run = private unnamed_addr constant [3 x i8] c"Run", align 1
@func.name = private unnamed_addr constant [10 x i8] c"func() int", align 1
@int.name = private unnamed_addr constant [3 x i8] c"int", align 1
@iface.name = private unnamed_addr constant [23 x i8] c"interface { Run() int }", align 1

@_llgo_main.Task = weak_odr constant { %"runtime/abi.StructType", %"runtime/abi.UncommonType", [2 x %"github.com/xgo-dev/llgo/runtime/abi.Method"] } {
  %"runtime/abi.StructType" {
    %"runtime/abi.Type" { i64 0, i64 0, i32 1, i8 13, i8 1, i8 1, i8 25, { ptr, ptr } { ptr @memequal0, ptr @_llgo_main.Task }, ptr null, %"runtime.String" { ptr @task.name, i64 9 }, ptr @"*_llgo_main.Task" },
    %"runtime.String" zeroinitializer,
    %"runtime.Slice" zeroinitializer
  },
  %"runtime/abi.UncommonType" { %"runtime.String" { ptr @task.pkg, i64 4 }, i16 2, i16 2, i32 24 },
  [2 x %"github.com/xgo-dev/llgo/runtime/abi.Method"] [
    %"github.com/xgo-dev/llgo/runtime/abi.Method" { %"runtime.String" { ptr @method.drop, i64 4 }, ptr @_llgo_func$run, ptr @"main.(*Task).Drop", ptr @"main.Task.Drop" },
    %"github.com/xgo-dev/llgo/runtime/abi.Method" { %"runtime.String" { ptr @method.run, i64 3 }, ptr @_llgo_func$run, ptr @"main.(*Task).Run", ptr @"main.Task.Run" }
  ]
}, align 8

@"*_llgo_main.Task" = weak_odr constant { %"runtime/abi.PtrType", %"runtime/abi.UncommonType", [2 x %"github.com/xgo-dev/llgo/runtime/abi.Method"] } {
  %"runtime/abi.PtrType" {
    %"runtime/abi.Type" { i64 8, i64 8, i32 2, i8 11, i8 8, i8 8, i8 54, { ptr, ptr } { ptr @memequalptr, ptr null }, ptr null, %"runtime.String" { ptr @task.name, i64 9 }, ptr null },
    ptr @_llgo_main.Task
  },
  %"runtime/abi.UncommonType" { %"runtime.String" { ptr @task.pkg, i64 4 }, i16 2, i16 2, i32 24 },
  [2 x %"github.com/xgo-dev/llgo/runtime/abi.Method"] [
    %"github.com/xgo-dev/llgo/runtime/abi.Method" { %"runtime.String" { ptr @method.drop, i64 4 }, ptr @_llgo_func$run, ptr @"main.(*Task).Drop", ptr @"main.(*Task).Drop" },
    %"github.com/xgo-dev/llgo/runtime/abi.Method" { %"runtime.String" { ptr @method.run, i64 3 }, ptr @_llgo_func$run, ptr @"main.(*Task).Run", ptr @"main.(*Task).Run" }
  ]
}, align 8

@_llgo_func$run = weak_odr constant %"runtime/abi.FuncType" {
  %"runtime/abi.Type" { i64 8, i64 8, i32 3, i8 0, i8 8, i8 8, i8 51, { ptr, ptr } zeroinitializer, ptr null, %"runtime.String" { ptr @func.name, i64 10 }, ptr @"*_llgo_func$run" },
  %"runtime.Slice" zeroinitializer,
  %"runtime.Slice" { ptr @"_llgo_func$run$out", i64 1, i64 1 }
}, align 8
@"*_llgo_func$run" = weak_odr constant %"runtime/abi.PtrType" {
  %"runtime/abi.Type" { i64 8, i64 8, i32 4, i8 10, i8 8, i8 8, i8 54, { ptr, ptr } { ptr @memequalptr, ptr null }, ptr null, %"runtime.String" { ptr @func.name, i64 10 }, ptr null },
  ptr @_llgo_func$run
}, align 8
@_llgo_int = weak_odr constant %"runtime/abi.Type" { i64 8, i64 0, i32 5, i8 12, i8 8, i8 8, i8 2, { ptr, ptr } { ptr @memequal64, ptr null }, ptr null, %"runtime.String" { ptr @int.name, i64 3 }, ptr @"*_llgo_int" }, align 8
@"*_llgo_int" = weak_odr constant %"runtime/abi.PtrType" {
  %"runtime/abi.Type" { i64 8, i64 8, i32 6, i8 10, i8 8, i8 8, i8 54, { ptr, ptr } { ptr @memequalptr, ptr null }, ptr null, %"runtime.String" { ptr @int.name, i64 3 }, ptr null },
  ptr @_llgo_int
}, align 8
@"_llgo_func$run$out" = weak_odr constant [1 x ptr] [ptr @_llgo_int], align 8

@_llgo_iface$Runner = weak_odr constant %"runtime/abi.InterfaceType" {
  %"runtime/abi.Type" { i64 16, i64 16, i32 7, i8 0, i8 8, i8 8, i8 20, { ptr, ptr } { ptr @interequal, ptr null }, ptr null, %"runtime.String" { ptr @iface.name, i64 23 }, ptr @"*_llgo_iface$Runner" },
  %"runtime.String" { ptr @task.pkg, i64 4 },
  %"runtime.Slice" { ptr @"_llgo_iface$Runner$imethods", i64 1, i64 1 }
}, align 8
@"*_llgo_iface$Runner" = weak_odr constant %"runtime/abi.PtrType" {
  %"runtime/abi.Type" { i64 8, i64 8, i32 8, i8 10, i8 8, i8 8, i8 54, { ptr, ptr } { ptr @memequalptr, ptr null }, ptr null, %"runtime.String" { ptr @iface.name, i64 23 }, ptr null },
  ptr @_llgo_iface$Runner
}, align 8
@"_llgo_iface$Runner$imethods" = weak_odr constant [1 x %"runtime/abi.Imethod"] [
  %"runtime/abi.Imethod" { %"runtime.String" { ptr @method.run, i64 3 }, ptr @_llgo_func$run }
], align 8

declare i1 @memequal0(ptr, ptr, ptr)
declare i1 @memequalptr(ptr, ptr, ptr)
declare i1 @memequal64(ptr, ptr, ptr)
declare i1 @interequal(ptr, ptr, ptr)
declare i64 @"main.Task.Drop"(%Task)
declare i64 @"main.Task.Run"(%Task)
declare i64 @"main.(*Task).Drop"(ptr)
declare i64 @"main.(*Task).Run"(ptr)
