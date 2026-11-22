package llvmattr

import (
	"github.com/xgo-dev/llgo/internal/env"
	"github.com/xgo-dev/llvm"
)

// LLVM 22 AllocFnKind bits. Keep the
// encodings named and check them against the LLVM IR parser in tests.
const (
	allocKindAlloc         uint64 = 1 << 0
	allocKindUninitialized uint64 = 1 << 3
	allocKindZeroed        uint64 = 1 << 4
)

const runtimePrefix = env.LLGoRuntimePkg + "/internal/runtime."

// A zero kind preserves AllocRoot's nonnull contract without authorizing
// removal of its uncollectable storage / explicit lifetime registration.
var allocatorKinds = map[string]uint64{
	runtimePrefix + "AllocU":       allocKindAlloc | allocKindUninitialized,
	runtimePrefix + "AllocZ":       allocKindAlloc | allocKindZeroed,
	runtimePrefix + "AllocAtomicU": allocKindAlloc | allocKindUninitialized,
	runtimePrefix + "AllocAtomicZ": allocKindAlloc | allocKindZeroed,
	runtimePrefix + "AllocRoot":    0,
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
	fn.AddAttributeAtIndex(0, ctx.CreateEnumAttribute(llvm.AttributeKindID("noundef"), 0))
	if kind != 0 {
		fn.AddFunctionAttr(ctx.CreateEnumAttribute(llvm.AttributeKindID("allocsize"), AllocSize(0, AllocSizeNoCount)))
	}
}

// ApplyAllocatorCall records the extent of positive constant allocations.
// Ordinary scanned and atomic allocations also return fresh storage.
func ApplyAllocatorCall(ctx llvm.Context, call, callee llvm.Value, args []llvm.Value, gcRoots bool) {
	if len(args) != 1 || callee.IsAFunction().IsNil() {
		return
	}
	kind, ok := allocatorKinds[callee.Name()]
	if !ok {
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
	call.AddCallSiteAttribute(0, ctx.CreateEnumAttribute(llvm.AttributeKindID("dereferenceable"), n))
	if kind == 0 {
		// Wasm AllocRoot publishes the backing allocation in an intrusive
		// root list and returns a pointer just past its header.
		return
	}
	call.AddCallSiteAttribute(0, ctx.CreateEnumAttribute(llvm.AttributeKindID("noalias"), 0))
	// Fresh storage is independent of whether the collector permits heap-to-stack.
	if !gcRoots {
		call.AddCallSiteAttribute(-1, ctx.CreateEnumAttribute(llvm.AttributeKindID("allockind"), kind))
	}
}
