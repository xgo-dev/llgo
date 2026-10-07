// Package llvmfmv implements early, CPU-guarded LLVM function specialization
// directly through the LLVM Go API, independently of optimization and LTO.
package llvmfmv

import (
	"fmt"
	"strings"

	"github.com/xgo-dev/llvm"
)

const (
	entryAttribute = "llgo.fmv.avx2-entry"
	doneAttribute  = "llgo.fmv.processed"
	inlineEntry    = "llgo.fmv.inline-entry"
	variantSuffix  = ".__llgo_fmv_avx2"
	baselineSuffix = ".__llgo_fmv_baseline"
	resolverSuffix = ".__llgo_fmv_resolve"
	slotSuffix     = ".__llgo_fmv_slot"
	queryAttribute = "llgo.cpu.query"
	avx2Query      = "x86.avx2"
	featureSymbol  = "github.com/xgo-dev/llgo/runtime/internal/runtime.CPUFeatures"
	// Keep these bits in sync with runtime/_patch/internal/cpu/init_llgo.go.
	featureInitialized = 1
	featureAVX2        = 2
)

func stringAttribute(fn llvm.Value, name string) string {
	if attr := fn.GetStringAttributeAtIndex(-1, name); !attr.IsNil() {
		return attr.GetStringValue()
	}
	return ""
}

func hasAttribute(fn llvm.Value, name string) bool {
	return !fn.GetStringAttributeAtIndex(-1, name).IsNil()
}

func isQuery(fn llvm.Value) bool {
	if fn.IsNil() || fn.IsAFunction().IsNil() || stringAttribute(fn, queryAttribute) != avx2Query {
		return false
	}
	typ := fn.GlobalValueType()
	ret := typ.ReturnType()
	if ret.TypeKind() != llvm.IntegerTypeKind || ret.IntTypeWidth() != 1 || typ.IsFunctionVarArg() {
		return false
	}
	// Before ABI lowering the Go method can still have its zero-sized receiver.
	params := typ.ParamTypes()
	return len(params) == 0 || len(params) == 1 && params[0].TypeKind() == llvm.StructTypeKind && params[0].StructElementTypesCount() == 0
}

// The first implementation preserves the existing SIMD128 ABI. Wider vectors
// and aggregates require their own target-independent boundary lowering.
func supportedType(typ llvm.Type) bool {
	switch typ.TypeKind() {
	case llvm.VoidTypeKind, llvm.IntegerTypeKind, llvm.PointerTypeKind,
		llvm.FloatTypeKind, llvm.DoubleTypeKind, llvm.X86_FP80TypeKind,
		llvm.FP128TypeKind, llvm.PPC_FP128TypeKind:
		return true
	case llvm.VectorTypeKind:
		elem := typ.ElementType()
		var width int
		switch elem.TypeKind() {
		case llvm.IntegerTypeKind:
			width = elem.IntTypeWidth()
		case llvm.FloatTypeKind:
			width = 32
		case llvm.DoubleTypeKind:
			width = 64
		default:
			return false
		}
		return typ.VectorSize() <= 128/width
	}
	return false
}

func supported(fn llvm.Value) bool {
	typ := fn.GlobalValueType()
	if typ.IsFunctionVarArg() || !supportedType(typ.ReturnType()) {
		return false
	}
	for _, name := range []string{"naked", "returns_twice"} {
		if !fn.GetEnumFunctionAttribute(llvm.AttributeKindID(name)).IsNil() {
			return false
		}
	}
	for _, typ := range typ.ParamTypes() {
		if !supportedType(typ) {
			return false
		}
	}
	return true
}

// Run preserves baseline entries and creates AVX2 implementations of supported
// SIMD128 functions. It runs after Win64 vector-parameter lowering but before
// aggregate ABI lowering and target optimization,
// including in O0 and ModeGen builds. baseline is the resolved compile-time
// GOAMD64 level, never a host environment read. Running it again is harmless.
func Run(mod llvm.Module, baseline string) error {
	arch, _, _ := strings.Cut(mod.Target(), "-")
	if arch != "x86_64" && arch != "amd64" && arch != "x86_64h" {
		return nil
	}
	if baseline != "" && baseline != "v1" && baseline != "v2" && baseline != "v3" && baseline != "v4" {
		return fmt.Errorf("SIMD FMV invalid GOAMD64 baseline: %s", baseline)
	}
	// Discover roots from direct uses of trusted queries. Ordinary packages
	// without SIMD must not pay for a scan of every instruction in the module.
	roots := make(map[llvm.Value]llvm.Value)
	for fn := mod.FirstFunction(); !fn.IsNil(); fn = llvm.NextFunction(fn) {
		if !isQuery(fn) {
			continue
		}
		for use := fn.FirstUse(); !use.IsNil(); use = use.NextUse() {
			call := use.User()
			if !call.IsACallInst().IsNil() && call.CalledValue() == fn {
				roots[call.InstructionParent().Parent()] = fn
			}
		}
	}
	if baseline == "v3" || baseline == "v4" {
		// The official CPU policy does not allow GODEBUG to disable features
		// required by GOAMD64. No version selection or extra helper entry is
		// needed once AVX2 is already part of the compile-time contract.
		for fn := range roots {
			specialize(mod.Context(), fn, nil, true)
		}
		return nil
	}
	var originals []llvm.Value
	for fn := mod.FirstFunction(); !fn.IsNil(); fn = llvm.NextFunction(fn) {
		if hasAttribute(fn, doneAttribute) {
			continue
		}
		if !hasAttribute(fn, entryAttribute) && roots[fn].IsNil() {
			continue
		}
		if !supported(fn) {
			continue
		}
		suffixes := []string{variantSuffix}
		if !roots[fn].IsNil() {
			suffixes = append(suffixes, baselineSuffix, resolverSuffix, slotSuffix)
		}
		for _, suffix := range suffixes {
			name := fn.Name() + suffix
			// Check all names before modifying any function in the module.
			if !mod.NamedValue(name).IsNil() {
				return fmt.Errorf("SIMD FMV symbol collision: %s", name)
			}
		}
		originals = append(originals, fn)
	}
	if len(originals) == 0 {
		return nil
	}
	if mask := mod.NamedValue(featureSymbol); !mask.IsNil() &&
		(mask.IsAGlobalVariable().IsNil() || mask.GlobalValueType() != mod.Context().Int64Type()) {
		return fmt.Errorf("SIMD FMV symbol collision: %s", featureSymbol)
	}

	ctx := mod.Context()
	source := newSourceInfo(mod)
	variants := make(map[llvm.Value]llvm.Value, len(originals))
	baselines := make(map[llvm.Value]llvm.Value, len(roots))
	for _, fn := range originals {
		variant := fn.CloneFunction()
		variant.SetName(fn.Name() + variantSuffix)
		if !fn.IsDeclaration() {
			source.clone(fn, variant)
		}
		features := stringAttribute(fn, "target-features")
		if features != "" {
			features += ","
		}
		variant.AddFunctionAttr(ctx.CreateStringAttribute("target-features", features+"+avx,+avx2"))
		variant.RemoveStringAttributeAtIndex(-1, entryAttribute)
		variant.RemoveStringAttributeAtIndex(-1, inlineEntry)
		variant.AddFunctionAttr(ctx.CreateStringAttribute(doneAttribute, ""))
		fn.AddFunctionAttr(ctx.CreateStringAttribute(doneAttribute, ""))
		variants[fn] = variant
		if !roots[fn].IsNil() {
			baseline := fn.CloneFunction()
			baseline.SetName(fn.Name() + baselineSuffix)
			baseline.SetLinkage(llvm.InternalLinkage)
			baseline.RemoveStringAttributeAtIndex(-1, entryAttribute)
			baseline.RemoveStringAttributeAtIndex(-1, inlineEntry)
			source.clone(fn, baseline)
			baselines[fn] = baseline
		}
	}
	for _, fn := range originals {
		if fn.IsDeclaration() {
			continue
		}
		variant := variants[fn]
		specialize(ctx, variant, variants, true)
		if baseline := baselines[fn]; !baseline.IsNil() {
			specialize(ctx, baseline, baselines, false)
			dispatch(mod, source, fn, baseline, variant)
		}
	}
	return nil
}

func specialize(ctx llvm.Context, fn llvm.Value, variants map[llvm.Value]llvm.Value, enabled bool) {
	var queries []llvm.Value
	for bb := fn.FirstBasicBlock(); !bb.IsNil(); bb = llvm.NextBasicBlock(bb) {
		for inst := bb.FirstInstruction(); !inst.IsNil(); inst = llvm.NextInstruction(inst) {
			if inst.IsACallInst().IsNil() && inst.IsAInvokeInst().IsNil() {
				continue
			}
			callee := inst.CalledValue()
			if isQuery(callee) {
				if !inst.IsACallInst().IsNil() {
					queries = append(queries, inst)
				}
			} else if variant := variants[callee]; !variant.IsNil() {
				inst.SetOperand(inst.OperandsCount()-1, variant)
			}
		}
	}
	for _, query := range queries {
		var bit uint64
		if enabled {
			bit = 1
		}
		query.ReplaceAllUsesWith(llvm.ConstInt(ctx.Int1Type(), bit, false))
		query.EraseFromParentAsInstruction()
	}
	// Mandatory legality cleanup, even at O0. These are local LLVM IR utilities,
	// not a registered pass or an optional optimization pipeline. Iterate for
	// conditions propagated through PHIs and short-circuit boolean expressions.
	for {
		changed := false
		for bb := fn.FirstBasicBlock(); !bb.IsNil(); bb = llvm.NextBasicBlock(bb) {
			if bb.SimplifyInstructionsInBlock() {
				changed = true
			}
			if bb.ConstantFoldTerminator(true) {
				changed = true
			}
		}
		if fn.RemoveUnreachableBlocks() {
			changed = true
		}
		if !changed {
			break
		}
	}
}

func dispatch(mod llvm.Module, source *sourceInfo, fn, baseline, variant llvm.Value) {
	ctx := mod.Context()
	resolver := fn.CloneFunction()
	resolver.SetName(fn.Name() + resolverSuffix)
	resolver.SetLinkage(llvm.InternalLinkage)
	resolver.RemoveStringAttributeAtIndex(-1, entryAttribute)
	resolver.RemoveStringAttributeAtIndex(-1, inlineEntry)
	resolver.RemoveEnumAttributeAtIndex(-1, llvm.AttributeKindID("alwaysinline"))
	resolver.AddFunctionAttr(ctx.CreateEnumAttribute(llvm.AttributeKindID("noinline"), 0))
	source.cloneEntry(fn, resolver)

	mask := mod.NamedGlobal(featureSymbol)
	if mask.IsNil() {
		mask = llvm.AddGlobal(mod, ctx.Int64Type(), featureSymbol)
		mask.SetAlignment(8)
	}
	slot := llvm.AddGlobal(mod, fn.Type(), fn.Name()+slotSuffix)
	slot.SetLinkage(llvm.InternalLinkage)
	slot.SetInitializer(resolver)
	slot.SetAlignment(8)

	b := ctx.NewBuilder()
	defer b.Dispose()
	b.SetInsertPointAtEnd(ctx.InsertBasicBlock(fn.EntryBasicBlock(), "fmv.entry"))
	// The ABI thunk is not another source execution frame. LLVM propagates the
	// real callsite (including its outer inline chain) when inlining a nodebug
	// callee. Keep source subprograms on the physical implementations and retain
	// the public funcinfo identity for function values and sampled entry PCs.
	// This gives the same synthetic-scope policy before and during LTO without
	// requiring a compiler-specific late pass in the native linker.
	fn.SetSubprogram(llvm.Metadata{})
	if hasAttribute(fn, inlineEntry) {
		fn.RemoveEnumAttributeAtIndex(-1, llvm.AttributeKindID("noinline"))
		fn.RemoveStringAttributeAtIndex(-1, inlineEntry)
	}
	target := b.CreateLoad(fn.Type(), slot, "target")
	target.SetAlignment(8)
	target.SetOrdering(llvm.AtomicOrderingMonotonic)
	tailReturn(b, fn, target)
	fn.RemoveUnreachableBlocks()

	old := resolver.EntryBasicBlock()
	entry := ctx.InsertBasicBlock(old, "fmv.resolve")
	uninitialized := ctx.InsertBasicBlock(old, "fmv.uninitialized")
	selectImpl := ctx.InsertBasicBlock(old, "fmv.select")
	b.SetInsertPointAtEnd(entry)
	if sp := resolver.Subprogram(); !sp.IsNil() {
		b.SetCurrentDebugLocation(0, 0, sp, llvm.Metadata{})
	}
	features := b.CreateLoad(ctx.Int64Type(), mask, "features")
	features.SetAlignment(8)
	initialized := b.CreateAnd(features, llvm.ConstInt(ctx.Int64Type(), featureInitialized, false), "")
	b.CreateCondBr(b.CreateICmp(llvm.IntNE, initialized, llvm.ConstInt(ctx.Int64Type(), 0, false), ""), selectImpl, uninitialized)
	b.SetInsertPointAtEnd(uninitialized)
	// Do not cache a pre-initialization fallback. Keep only this edge opaque:
	// the resolver must remain tail-only, while ordinary callers may inline
	// either implementation when their target features are compatible.
	call := tailReturn(b, resolver, baseline)
	call.AddCallSiteAttribute(-1, ctx.CreateEnumAttribute(llvm.AttributeKindID("noinline"), 0))
	b.SetInsertPointAtEnd(selectImpl)
	enabled := b.CreateAnd(features, llvm.ConstInt(ctx.Int64Type(), featureAVX2, false), "")
	check := b.CreateICmp(llvm.IntNE, enabled, llvm.ConstInt(ctx.Int64Type(), 0, false), "")
	selected := b.CreateSelect(check, variant, baseline, "selected")
	publish := b.CreateStore(selected, slot)
	publish.SetAlignment(8)
	publish.SetOrdering(llvm.AtomicOrderingMonotonic)
	tailReturn(b, resolver, selected)
	resolver.RemoveUnreachableBlocks()
}

func tailReturn(b llvm.Builder, fn, target llvm.Value) llvm.Value {
	call := b.CreateCall(fn.GlobalValueType(), target, fn.Params(), "")
	call.SetInstructionCallConv(fn.FunctionCallConv())
	// Forward parameter/return ABI attributes, not function optimization policy.
	for index := 0; index <= fn.ParamsCount(); index++ {
		for _, attr := range fn.GetAttributesAtIndex(index) {
			call.AddCallSiteAttribute(index, attr)
		}
	}
	call.SetTailCallKind(llvm.TailCallKindMustTail)
	if fn.GlobalValueType().ReturnType().TypeKind() == llvm.VoidTypeKind {
		b.CreateRetVoid()
	} else {
		b.CreateRet(call)
	}
	return call
}
