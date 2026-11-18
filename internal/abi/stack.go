package abi

import runtimeabi "github.com/xgo-dev/llgo/runtime/abi"

const SmallStackBufferSize = runtimeabi.SmallStackBufferSize

// MaxEscapeStackBytes bounds storage newly promoted by Go escape analysis and
// its lowering helpers in one function. Existing locals and later LLVM frame
// layout remain independent; this is not a bound on the entire call stack.
const MaxEscapeStackBytes uint64 = 64 * 1024
