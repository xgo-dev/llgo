; ModuleID = 'dst'
source_filename = "dst"

%"github.com/xgo-dev/llgo/runtime/abi.Method" = type { %runtime.String, { ptr, i32 }, { ptr, i32 }, { ptr, i32 } }
%runtime.String = type { { ptr, i32 }, i64 }

@_llgo_main.Task = constant { i32, [2 x %"github.com/xgo-dev/llgo/runtime/abi.Method"] } { i32 2, [2 x %"github.com/xgo-dev/llgo/runtime/abi.Method"] [%"github.com/xgo-dev/llgo/runtime/abi.Method" { %runtime.String { { ptr, i32 } { ptr @0, i32 0 }, i64 4 }, { ptr, i32 } { ptr @method.type, i32 0 }, { ptr, i32 } { ptr @"github.com/xgo-dev/llgo/runtime/internal/runtime.unreachableMethod", i32 0 }, { ptr, i32 } { ptr @"github.com/xgo-dev/llgo/runtime/internal/runtime.unreachableMethod", i32 0 } }, %"github.com/xgo-dev/llgo/runtime/abi.Method" { %runtime.String { { ptr, i32 } { ptr @1, i32 0 }, i64 3 }, { ptr, i32 } { ptr @method.type, i32 0 }, { ptr, i32 } { ptr @Run, i32 0 }, { ptr, i32 } { ptr @Run, i32 0 } }] }, align 8
@0 = private constant [4 x i8] c"Drop"
@method.type = external global i8
@1 = private constant [3 x i8] c"Run"

declare void @"github.com/xgo-dev/llgo/runtime/internal/runtime.unreachableMethod"()

declare void @Run()
