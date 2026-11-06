// Package llvmattr describes compiler-owned LLVM contracts for runtime allocators.
package llvmattr

import (
	"github.com/xgo-dev/llgo/internal/env"
	"github.com/xgo-dev/llvm"
)

// LLVM 22 AllocFnKind bits and the absent second allocsize argument. Keep the
// encodings named and check them against the LLVM IR parser in tests.
const (
	allocKindAlloc         uint64 = 1 << 0
	allocKindUninitialized uint64 = 1 << 3
	allocKindZeroed        uint64 = 1 << 4
	allocSizeBytes         uint64 = uint64(^uint32(0)) // allocsize(0)
)

const runtimePrefix = env.LLGoRuntimePkg + "/internal/runtime."

// A zero kind preserves AllocRoot's nonnull contract without authorizing
// removal of its uncollectable storage / explicit lifetime registration.
var allocatorKinds = map[string]uint64{
	runtimePrefix + "AllocU":    allocKindAlloc | allocKindUninitialized,
	runtimePrefix + "AllocZ":    allocKindAlloc | allocKindZeroed,
	runtimePrefix + "AllocRoot": 0,
	// Preserve the existing nonnull contract without extending placement policy.
	runtimePrefix + "AllocNoScanU": 0,
	runtimePrefix + "AllocNoScanZ": 0,
}

// ApplyAllocatorFunction is shared by SSA declarations and late ABI lowering.
// The raw allocators return shared zerobase for zero bytes, so fresh-allocation
// properties belong on proven-positive calls, not on these declarations.
func ApplyAllocatorFunction(ctx llvm.Context, fn llvm.Value) {
	kind, ok := allocatorKinds[fn.Name()]
	if !ok {
		return
	}
	fn.AddAttributeAtIndex(0, ctx.CreateEnumAttribute(llvm.AttributeKindID("nonnull"), 0))
	if kind != 0 {
		fn.AddFunctionAttr(ctx.CreateEnumAttribute(llvm.AttributeKindID("allocsize"), allocSizeBytes))
	}
}

// ApplyAllocatorCall enables allocation elision and Attributor heap-to-stack
// for positive constant-size AllocU/AllocZ calls. Unknown sizes and zero-sized
// sentinel identities retain their original behavior. Sizes above the signed
// maximum of the argument type are not treated as proven positive.
// Explicit-root collectors cannot scan pointer fields in newly introduced
// stack storage, so they keep
// their heap allocations as well.
func ApplyAllocatorCall(ctx llvm.Context, call, callee llvm.Value, args []llvm.Value, gcRoots bool) {
	if gcRoots || len(args) != 1 || callee.IsAFunction().IsNil() {
		return
	}
	kind := allocatorKinds[callee.Name()]
	if kind == 0 {
		return
	}
	size := args[0].IsAConstantInt()
	if size.IsNil() {
		return
	}
	width := size.Type().IntTypeWidth()
	if width < 2 || width > 64 {
		return
	}
	n := size.ZExtValue()
	maxSize := (uint64(1) << (width - 1)) - 1
	if n == 0 || n > maxSize {
		return
	}
	call.AddCallSiteAttribute(-1, ctx.CreateEnumAttribute(llvm.AttributeKindID("allockind"), kind))
}
