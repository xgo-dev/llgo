; ModuleID = 'dst'
source_filename = "dst"

%"runtime/abi.StructType" = type { %"runtime/abi.Type", %runtime.String, %runtime.Slice }
%"runtime/abi.Type" = type { i64, i64, i32, i8, i8, i8, i8, { ptr, ptr }, ptr, %runtime.String, ptr }
%runtime.String = type { ptr, i64 }
%runtime.Slice = type { ptr, i64, i64 }
%"runtime/abi.UncommonType" = type { %runtime.String, i16, i16, i32 }
%"github.com/xgo-dev/llgo/runtime/abi.Method" = type { %runtime.String, ptr, ptr, ptr }
%"runtime/abi.PtrType" = type { %"runtime/abi.Type", ptr }
%"runtime/abi.FuncType" = type { %"runtime/abi.Type", %runtime.Slice, %runtime.Slice }
%Task = type opaque

@_llgo_main.Task = constant { %"runtime/abi.StructType", %"runtime/abi.UncommonType", [2 x %"github.com/xgo-dev/llgo/runtime/abi.Method"] } { %"runtime/abi.StructType" { %"runtime/abi.Type" { i64 0, i64 0, i32 1, i8 13, i8 1, i8 1, i8 25, { ptr, ptr } { ptr @memequal0, ptr @_llgo_main.Task }, ptr null, %runtime.String { ptr @0, i64 9 }, ptr @"*_llgo_main.Task" }, %runtime.String zeroinitializer, %runtime.Slice zeroinitializer }, %"runtime/abi.UncommonType" { %runtime.String { ptr @1, i64 4 }, i16 2, i16 2, i32 24 }, [2 x %"github.com/xgo-dev/llgo/runtime/abi.Method"] [%"github.com/xgo-dev/llgo/runtime/abi.Method" { %runtime.String { ptr @2, i64 4 }, ptr @"_llgo_func$run", ptr @"github.com/xgo-dev/llgo/runtime/internal/runtime.unreachableMethod", ptr @"github.com/xgo-dev/llgo/runtime/internal/runtime.unreachableMethod" }, %"github.com/xgo-dev/llgo/runtime/abi.Method" { %runtime.String { ptr @3, i64 3 }, ptr @"_llgo_func$run", ptr @"main.(*Task).Run", ptr @main.Task.Run }] }, align 8
@0 = private constant [9 x i8] c"main.Task", align 1
@"*_llgo_main.Task" = constant { %"runtime/abi.PtrType", %"runtime/abi.UncommonType", [2 x %"github.com/xgo-dev/llgo/runtime/abi.Method"] } { %"runtime/abi.PtrType" { %"runtime/abi.Type" { i64 8, i64 8, i32 2, i8 11, i8 8, i8 8, i8 54, { ptr, ptr } { ptr @memequalptr, ptr null }, ptr null, %runtime.String { ptr @0, i64 9 }, ptr null }, ptr @_llgo_main.Task }, %"runtime/abi.UncommonType" { %runtime.String { ptr @1, i64 4 }, i16 2, i16 2, i32 24 }, [2 x %"github.com/xgo-dev/llgo/runtime/abi.Method"] [%"github.com/xgo-dev/llgo/runtime/abi.Method" { %runtime.String { ptr @2, i64 4 }, ptr @"_llgo_func$run", ptr @"github.com/xgo-dev/llgo/runtime/internal/runtime.unreachableMethod", ptr @"github.com/xgo-dev/llgo/runtime/internal/runtime.unreachableMethod" }, %"github.com/xgo-dev/llgo/runtime/abi.Method" { %runtime.String { ptr @3, i64 3 }, ptr @"_llgo_func$run", ptr @"main.(*Task).Run", ptr @"main.(*Task).Run" }] }, align 8
@1 = private constant [4 x i8] c"main", align 1
@2 = private constant [4 x i8] c"Drop", align 1
@"_llgo_func$run" = external global %"runtime/abi.FuncType"
@3 = private constant [3 x i8] c"Run", align 1

declare i1 @memequal0(ptr, ptr, ptr)

declare void @"github.com/xgo-dev/llgo/runtime/internal/runtime.unreachableMethod"()

declare i64 @"main.(*Task).Run"(ptr)

declare i64 @main.Task.Run(%Task)

declare i1 @memequalptr(ptr, ptr, ptr)
