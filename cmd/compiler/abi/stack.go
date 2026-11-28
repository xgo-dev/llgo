package abi

import "github.com/xgo-dev/llgo/cmd/compiler/abipolicy"

const SmallStackBufferSize = abipolicy.SmallStackBufferSize

// MaxEscapeStackBytes bounds storage newly promoted by Go escape analysis and
// its lowering helpers in one function. Existing locals and later LLVM frame
// layout remain independent; this is not a bound on the entire call stack.
const MaxEscapeStackBytes = abipolicy.MaxEscapeStackBytes
