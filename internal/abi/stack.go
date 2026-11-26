package abi

import runtimeabi "github.com/xgo-dev/llgo/runtime/abi"

const SmallStackBufferSize = runtimeabi.SmallStackBufferSize

// FiberStackOverhead mirrors the runtime module's fiber stack contract: how
// much of a fixed-size fiber goroutine stack is unavailable to caller-owned
// backing arrays and large locals (guard margin, entry chain, fixed frames).
// The stack-residency budgets cap at stack size minus this overhead.
const FiberStackOverhead = runtimeabi.FiberStackOverhead

// MaxEscapeStackBytes bounds storage newly promoted by Go escape analysis and
// its lowering helpers in one function. Existing locals and later LLVM frame
// layout remain independent; this is not a bound on the entire call stack.
const MaxEscapeStackBytes uint64 = 64 * 1024
