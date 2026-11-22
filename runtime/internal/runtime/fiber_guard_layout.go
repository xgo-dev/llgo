//go:build llgo && baremetal && tinygo.riscv32 && !nogc

package runtime

import (
	"unsafe"

	abi "github.com/xgo-dev/llgo/runtime/abi"
)

// Compile-time layout contracts for the fiber swap guard. The asm in
// _wrap/context_riscv32.c hardcodes baremetalFiberContext's field offsets
// and the swap-frame geometry, so the contracts are enforced where every
// fiber-target build typechecks them. Each declaration is an array bound
// that must stay non-negative, so any drift fails the build instead of the
// firmware's first swap.
//
// This package has no runnable `go test` under the fiber tags (the hosted
// clite declarations conflict at build, and the C linknames cannot resolve
// in a native test binary), which is why the contracts live as compile-time
// assertions rather than test assertions.
var (
	// baremetalFiberContext: sp@0, stackLo@4, stackHi@8 (uintptr is 4
	// bytes on RV32; the asm reads stackLo at offset 4).
	_ [unsafe.Offsetof(baremetalFiberContext{}.sp)]struct{}
	_ [unsafe.Offsetof(baremetalFiberContext{}.stackLo) - 4]struct{}
	_ [4 - unsafe.Offsetof(baremetalFiberContext{}.stackLo)]struct{}
	_ [unsafe.Offsetof(baremetalFiberContext{}.stackHi) - 8]struct{}
	_ [8 - unsafe.Offsetof(baremetalFiberContext{}.stackHi)]struct{}

	// Swap-frame geometry shared with llgoFiberSwap: 64-byte frame, ra at
	// the base, the a0 argument slot inside the frame.
	_ [64 - fiberFrameSize]struct{}
	_ [fiberFrameSize - 64]struct{}
	_ [0 - fiberFrameRAOff]struct{}
	_ [52 - fiberFrameArgOff]struct{}
	_ [fiberFrameArgOff - 52]struct{}

	// The canary must never be zero (zero is the allocator's fill), and the
	// minimum stack size must always cover the guard margin (68) plus the
	// initial swap frame (64) plus the worst-case 16-byte alignment
	// round-down (15): 147.
	_ [1 / fiberStackCanary]struct{}
	_ [fiberMinStackSize - 147]struct{}

	// The Go-side make-slice guard must mirror llgoFiberGuardMargin (68 =
	// 64-byte swap frame + 4-byte canary word) in _wrap/context_riscv32.c.
	_ [68 - fiberStackGuardMargin]struct{}
	_ [fiberStackGuardMargin - 68]struct{}

	// The compiler's fixed-stack budget (stack size minus abi.FiberStackOverhead)
	// must leave at least the guard margin plus the initial swap frame above
	// the floor, or a budget-exact array would park on the canary.
	_ [abi.FiberStackOverhead - (fiberStackGuardMargin + fiberFrameSize)]struct{}
)
