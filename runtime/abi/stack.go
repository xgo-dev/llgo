package abi

// SmallStackBufferSize is the shared compiler/runtime contract for caller-owned
// temporary buffers: bytes for slices/strings, rune slots for UTF-8 decoding.
const SmallStackBufferSize = 32

// FiberStackOverhead is the shared compiler/runtime contract for how much of
// a fixed-size fiber goroutine stack is unavailable to caller-owned backing
// arrays and large locals: the swap guard margin (68: 64-byte swap frame +
// 4-byte floor canary), the fiber entry chain, and each frame's fixed
// prologue. The stack-residency budgets cap at stack size minus this
// overhead, so a budget-exact array cannot consume the whole stack before
// the entry chain and swap frame fit. Keep in sync with fiberStackGuardMargin
// and fiberMinStackSize in runtime/internal/runtime; the runtime pins the
// split with compile-time assertions in fiber_guard_layout.go.
const FiberStackOverhead = 256
