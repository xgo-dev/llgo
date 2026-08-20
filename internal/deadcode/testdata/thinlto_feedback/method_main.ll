target triple = "x86_64-unknown-linux-gnu"

%"github.com/goplus/llgo/runtime/abi.Method" = type { ptr, ptr, ptr, ptr }
%T.type = type { ptr, [2 x %"github.com/goplus/llgo/runtime/abi.Method"] }
@T = external hidden constant %T.type
@flag = constant i1 false

define i32 @main() {
entry:
  call void @liveDemand()
  %enabled = load i1, ptr @flag
  br i1 %enabled, label %demand, label %done
demand:
  call void @semanticDemand()
  br label %done
done:
  ret i32 0
}

declare hidden void @liveDemand()
declare hidden void @semanticDemand()
