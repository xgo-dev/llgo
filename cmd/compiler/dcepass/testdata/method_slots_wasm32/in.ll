target datalayout = "e-m:e-p:32:32-i64:64-n32:64-S128"
target triple = "wasm32-unknown-wasi"

%runtime.String = type { { ptr, i32 }, i64 }
%"github.com/xgo-dev/llgo/runtime/abi.Method" = type { %runtime.String, { ptr, i32 }, { ptr, i32 }, { ptr, i32 } }

@drop.name = private constant [4 x i8] c"Drop"
@run.name = private constant [3 x i8] c"Run"
@method.type = external constant i8

; The dropped slot shares TFn with the live slot. Rewriting it must not change
; other references to Run, even when the constants are uniqued by LLVM.
@_llgo_main.Task = weak_odr constant { i32, [2 x %"github.com/xgo-dev/llgo/runtime/abi.Method"] } {
  i32 2,
  [2 x %"github.com/xgo-dev/llgo/runtime/abi.Method"] [
    %"github.com/xgo-dev/llgo/runtime/abi.Method" { %runtime.String { { ptr, i32 } { ptr @drop.name, i32 0 }, i64 4 }, { ptr, i32 } { ptr @method.type, i32 0 }, { ptr, i32 } { ptr @Drop, i32 0 }, { ptr, i32 } { ptr @Run, i32 0 } },
    %"github.com/xgo-dev/llgo/runtime/abi.Method" { %runtime.String { { ptr, i32 } { ptr @run.name, i32 0 }, i64 3 }, { ptr, i32 } { ptr @method.type, i32 0 }, { ptr, i32 } { ptr @Run, i32 0 }, { ptr, i32 } { ptr @Run, i32 0 } }
  ]
}, align 8

declare void @Drop()
declare void @Run()
