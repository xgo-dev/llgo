// Package llvmfmv implements early, CPU-guarded LLVM function specialization
// directly through the LLVM Go API, independently of optimization and LTO.
package llvmfmv

import (
	"fmt"
	"strings"

	"github.com/xgo-dev/llvm"
)

const (
	attributePrefix = "llgo.fmv."
	symbolPrefix    = ".__llgo_fmv_"

	// Shared with the frontend that emits the FMV attributes and CPU queries.
	EntryAttribute       = attributePrefix + "avx2-entry"
	InlineEntryAttribute = attributePrefix + "inline-entry"
	QueryAttribute       = "llgo.cpu.query"
	AVX2Query            = "x86.avx2"

	doneAttribute  = attributePrefix + "processed"
	variantSuffix  = symbolPrefix + "avx2"
	baselineSuffix = symbolPrefix + "baseline"
	resolverSuffix = symbolPrefix + "resolve"
	slotSuffix     = symbolPrefix + "slot"
	featureSymbol  = "github.com/xgo-dev/llgo/runtime/internal/runtime.CPUFeatures"
	// Keep these bits in sync with runtime/_patch/internal/cpu/init_llgo.go.
	featureInitialized = 1
	moduleDone         = "llgo.fmv.done"
	featureAVX         = 4
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
	return checkedQuery(fn) == AVX2Query
}
func checkedQuery(fn llvm.Value) string {
	if fn.IsNil() || fn.IsAFunction().IsNil() {
		return ""
	}
	name := stringAttribute(fn, QueryAttribute)
	if name == "" {
		return ""
	}
	typ := fn.GlobalValueType()
	ret := typ.ReturnType()
	if ret.TypeKind() != llvm.IntegerTypeKind || ret.IntTypeWidth() != 1 || typ.IsFunctionVarArg() {
		return ""
	}
	params := typ.ParamTypes()
	if len(params) == 0 {
		return name
	}
	if len(params) != 1 {
		return ""
	}
	if x86ABIFor(fn.GlobalParent()).queryReceiver(params[0]) {
		return name
	}
	return ""
}

// ABI lowering leaves register tuples and single-leaf aggregate wrappers on
// some targets. Their leaves must still satisfy the supported SIMD width boundary.
func supportedType(typ llvm.Type) bool {
	switch typ.TypeKind() {
	case llvm.VoidTypeKind, llvm.IntegerTypeKind, llvm.PointerTypeKind,
		llvm.FloatTypeKind, llvm.DoubleTypeKind, llvm.X86_FP80TypeKind,
		llvm.FP128TypeKind, llvm.PPC_FP128TypeKind:
		return true
	case llvm.StructTypeKind:
		if typ.IsStructOpaque() {
			return false
		}
		for _, elem := range typ.StructElementTypes() {
			if !supportedType(elem) {
				return false
			}
		}
		return true
	case llvm.ArrayTypeKind:
		return supportedType(typ.ElementType())
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
		return typ.VectorSize() <= 512/width
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

// Run folds x86 CPU predicates guaranteed by the resolved GOAMD64 baseline
// and specializes AVX2 entries and guarded SIMD work. It runs after ABI
// lowering and before target optimization, including in O0 and ModeGen builds.
// Mark the module after both guard lowering and wide ABI lowering succeed.
func Run(mod llvm.Module, baseline string) error {
	arch, _, _ := strings.Cut(mod.Target(), "-")
	if arch != "x86_64" && arch != "amd64" && arch != "x86_64h" {
		return nil
	}
	if mod.NamedMetadataNumOperands(moduleDone) != 0 {
		return nil
	}
	if mask := mod.NamedValue(featureSymbol); !mask.IsNil() &&
		(mask.IsAGlobalVariable().IsNil() || mask.GlobalValueType() != mod.Context().Int64Type()) {
		return fmt.Errorf("SIMD FMV symbol collision: %s", featureSymbol)
	}
	var avx2 *avx2Plan
	switch baseline {
	case "v3", "v4":
		// AVX2 is guaranteed. Only optional features need dispatch.
	case "", "v1", "v2":
		var err error
		avx2, err = stageAVX2(mod)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("SIMD FMV invalid GOAMD64 baseline: %s", baseline)
	}
	if err := runGuardedFeatures(mod, avx2, baseline); err != nil {
		if avx2 != nil {
			for _, fn := range avx2.originals {
				for _, copy := range []llvm.Value{avx2.variants[fn], avx2.baselines[fn]} {
					if !copy.IsNil() {
						copy.ReplaceAllUsesWith(llvm.Undef(copy.Type()))
						copy.EraseFromParentAsFunction()
					}
				}
			}
		}
		return err
	}
	applyWideABI(mod, baseline)
	mod.AddNamedMetadataOperand(moduleDone, mod.Context().MDNode(nil))
	return nil
}

// Stage private AVX2 bodies so the guard pass can discover collisions on the
// simplified call graph before editing public entries or source metadata.
type avx2Plan struct {
	originals           []llvm.Value
	variants, baselines map[llvm.Value]llvm.Value
	origins             map[llvm.Value]llvm.Value
}

func stageAVX2(mod llvm.Module) (*avx2Plan, error) {
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
	var originals []llvm.Value
	for fn := mod.FirstFunction(); !fn.IsNil(); fn = llvm.NextFunction(fn) {
		if hasAttribute(fn, doneAttribute) {
			continue
		}
		if !hasAttribute(fn, EntryAttribute) && roots[fn].IsNil() {
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
				return nil, fmt.Errorf("SIMD FMV symbol collision: %s", name)
			}
		}
		originals = append(originals, fn)
	}
	if len(originals) == 0 {
		return nil, nil
	}

	ctx := mod.Context()
	origins := make(map[llvm.Value]llvm.Value)
	variants := make(map[llvm.Value]llvm.Value, len(originals))
	baselines := make(map[llvm.Value]llvm.Value, len(roots))
	for _, fn := range originals {
		variant := fn.CloneFunction()
		variant.SetName(fn.Name() + variantSuffix)
		origins[variant] = fn
		features := stringAttribute(fn, "target-features")
		if features != "" {
			features += ","
		}
		variant.AddFunctionAttr(ctx.CreateStringAttribute("target-features", features+"+avx,+avx2"))
		variant.RemoveStringAttributeAtIndex(-1, EntryAttribute)
		variant.RemoveStringAttributeAtIndex(-1, InlineEntryAttribute)
		variant.AddFunctionAttr(ctx.CreateStringAttribute(doneAttribute, ""))
		variants[fn] = variant
		if !roots[fn].IsNil() {
			baseline := fn.CloneFunction()
			baseline.SetName(fn.Name() + baselineSuffix)
			baseline.SetLinkage(llvm.InternalLinkage)
			baseline.RemoveStringAttributeAtIndex(-1, EntryAttribute)
			baseline.RemoveStringAttributeAtIndex(-1, InlineEntryAttribute)
			baseline.AddFunctionAttr(ctx.CreateStringAttribute(doneAttribute, ""))
			origins[baseline] = fn
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
		}
	}
	return &avx2Plan{originals, variants, baselines, origins}, nil
}

func (p *avx2Plan) commit(mod llvm.Module) []dispatchSite {
	if p == nil {
		return nil
	}
	source := newSourceInfo(mod)
	var sites []dispatchSite
	for _, fn := range p.originals {
		fn.AddFunctionAttr(mod.Context().CreateStringAttribute(doneAttribute, ""))
		variant := p.variants[fn]
		if !fn.IsDeclaration() {
			// CloneFunction preserves linkage but drops COMDAT. Give the
			// variant its own group so generic instances merge independently
			// of their original definitions when linking COFF objects.
			if original := fn.Comdat(); original.C != nil {
				group := mod.Comdat(variant.Name())
				group.SetSelectionKind(original.SelectionKind())
				variant.SetComdat(group)
			}
			source.clone(fn, variant)
		}
		if baseline := p.baselines[fn]; !baseline.IsNil() {
			source.clone(fn, baseline)
			sites = append(sites, dispatch(mod, source, fn, baseline, []dispatchChoice{{bits: featureAVX2, fn: variant}}))
		}
	}
	return sites
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
	simplifyGuards(fn)
}

func simplifyGuards(fn llvm.Value) {
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

type dispatchChoice struct {
	bits   uint64
	fn     llvm.Value
	anyBit bool // alternative predicates can guarantee the same register width
}

// The AVX2 resolver has one select between its two implementations. If
// either gets further variants, select its physical body here as well.
type dispatchSite struct {
	features, selection, uninitializedCall llvm.Value
}

func selectDispatchBody(b llvm.Builder, features, baseline llvm.Value, choices []dispatchChoice) llvm.Value {
	selected := baseline
	for _, choice := range choices {
		required := llvm.ConstInt(features.Type(), choice.bits, false)
		enabled := b.CreateAnd(features, required, "")
		op, compare := llvm.IntEQ, required
		if choice.anyBit {
			op, compare = llvm.IntNE, llvm.ConstInt(features.Type(), 0, false)
		}
		check := b.CreateICmp(op, enabled, compare, "")
		selected = b.CreateSelect(check, choice.fn, selected, "selected")
	}
	return selected
}

func dispatch(mod llvm.Module, source *sourceInfo, fn, baseline llvm.Value, choices []dispatchChoice) dispatchSite {
	ctx := mod.Context()
	resolver := fn.CloneFunction()
	resolver.SetName(fn.Name() + resolverSuffix)
	resolver.SetLinkage(llvm.InternalLinkage)
	resolver.RemoveStringAttributeAtIndex(-1, EntryAttribute)
	resolver.RemoveStringAttributeAtIndex(-1, InlineEntryAttribute)
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
	if hasAttribute(fn, InlineEntryAttribute) {
		fn.RemoveEnumAttributeAtIndex(-1, llvm.AttributeKindID("noinline"))
		fn.RemoveStringAttributeAtIndex(-1, InlineEntryAttribute)
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
	// internal/cpu publishes the mask once before initializing its consumers.
	// Pre-publication bootstrap calls are sequential and must not cache their
	// fallback. After publication the mask is immutable; only the per-function
	// slot can be updated concurrently and therefore needs atomic accesses.
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
	selected := selectDispatchBody(b, features, baseline, choices)
	publish := b.CreateStore(selected, slot)
	publish.SetAlignment(8)
	publish.SetOrdering(llvm.AtomicOrderingMonotonic)
	tailReturn(b, resolver, selected)
	resolver.RemoveUnreachableBlocks()
	return dispatchSite{features, selected, call}
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
	call.SetTailCallKind(x86ABIFor(fn.GlobalParent()).forwardingKind(fn))
	if fn.GlobalValueType().ReturnType().TypeKind() == llvm.VoidTypeKind {
		b.CreateRetVoid()
	} else {
		b.CreateRet(call)
	}
	return call
}
