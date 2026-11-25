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
	PreferFeatureMD      = attributePrefix + "prefer-feature"
	AVX2Query            = "x86.avx2"

	moduleDone     = "llgo.fmv.done"
	variantSuffix  = symbolPrefix + "avx2"
	baselineSuffix = symbolPrefix + "baseline"
	resolverSuffix = symbolPrefix + "resolve"
	slotSuffix     = symbolPrefix + "slot"
	featureSymbol  = "github.com/xgo-dev/llgo/runtime/internal/runtime.CPUFeatures"
	// Keep these bits in sync with runtime/_patch/internal/cpu/init_llgo.go.
	featureInitialized = 1
	featureAVX2        = 2
	featureAVX         = 4
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
// baseline is the resolved compilation target, independent of the host CPU.
// A successful run marks the module after wide ABI lowering. Later calls
// skip the module, including its generated dispatchers and implementations.
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
	abi := newWideABI(mod)
	defer abi.data.Dispose()
	var avx2 *avx2Plan
	switch baseline {
	case "v3", "v4":
		// AVX2 is guaranteed. Only optional features need dispatch.
	case "", "v1", "v2":
		var err error
		avx2, err = planAVX2(abi)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("SIMD FMV invalid GOAMD64 baseline: %s", baseline)
	}
	if err := planVersions(abi, avx2, baseline); err != nil {
		if avx2 != nil {
			for _, fn := range avx2.originals {
				discardStagedBody(avx2.variants[fn])
				discardStagedBody(avx2.baselines[fn])
			}
		}
		return err
	}
	mod.AddNamedMetadataOperand(moduleDone, mod.Context().MDNode(nil))
	return nil
}

func discardStagedBody(body llvm.Value) {
	if !body.IsNil() {
		body.ReplaceAllUsesWith(llvm.Undef(body.Type()))
		body.EraseFromParentAsFunction()
	}
}

// Canonical AVX2 entries are part of the cross-package call contract. Plan
// their names on the original module, including declarations implemented elsewhere.
type avx2Plan struct {
	originals           []llvm.Value
	roots               map[llvm.Value]bool
	variants, baselines map[llvm.Value]llvm.Value
	origins             map[llvm.Value]llvm.Value
}

func planAVX2(abi *wideABI) (*avx2Plan, error) {
	mod := abi.mod
	// Discover roots from direct uses of trusted queries. Ordinary packages
	// without SIMD must not pay for a scan of every instruction in the module.
	queries := make(map[llvm.Value]uint64)
	roots := make(map[llvm.Value]bool)
	entries := make(map[llvm.Value]bool)
	for fn := mod.FirstFunction(); !fn.IsNil(); fn = llvm.NextFunction(fn) {
		if hasAttribute(fn, EntryAttribute) && supported(fn) {
			entries[fn] = true
		}
		if !isQuery(fn) {
			continue
		}
		queries[fn] = featureAVX2
		for use := fn.FirstUse(); !use.IsNil(); use = use.NextUse() {
			call := use.User()
			if !call.IsACallInst().IsNil() && call.CalledValue() == fn {
				roots[call.InstructionParent().Parent()] = true
			}
		}
	}
	for fn := range roots {
		if !supported(fn) || !abi.needsAVX2Entry(fn, queries, entries) {
			delete(roots, fn)
		}
	}
	var originals []llvm.Value
	for fn := mod.FirstFunction(); !fn.IsNil(); fn = llvm.NextFunction(fn) {
		if !entries[fn] && !roots[fn] {
			continue
		}
		suffixes := []string{variantSuffix}
		if roots[fn] {
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

	return &avx2Plan{originals: originals, roots: roots}, nil
}

// Canonical dispatch is needed for a direct AVX2 guard over local vector
// work or a call to a promised entry. Pointer helper chains use the normal
// guard planner.
func (a *wideABI) needsAVX2Entry(fn llvm.Value, queries map[llvm.Value]uint64, entries map[llvm.Value]bool) bool {
	for block, facts := range blockGuards(fn, queries) {
		if facts.common&featureAVX2 == 0 {
			continue
		}
		for inst := block.FirstInstruction(); !inst.IsNil(); inst = llvm.NextInstruction(inst) {
			if a.localVectorWork(inst) || entries[directGuardCallee(inst)] {
				return true
			}
		}
	}
	return false
}

// Prepare the two constant-folded views before emitting dispatch. Even an
// imported declaration can supply a canonical AVX2 entry.
func (p *avx2Plan) materialize() {
	if p == nil {
		return
	}
	ctx := p.originals[0].GlobalParent().Context()
	p.origins = make(map[llvm.Value]llvm.Value)
	p.variants = make(map[llvm.Value]llvm.Value, len(p.originals))
	p.baselines = make(map[llvm.Value]llvm.Value, len(p.roots))
	for _, fn := range p.originals {
		variant := fn.CloneFunction()
		variant.SetName(fn.Name() + variantSuffix)
		p.origins[variant] = fn
		features := stringAttribute(fn, "target-features")
		if features != "" {
			features += ","
		}
		variant.AddFunctionAttr(ctx.CreateStringAttribute("target-features", features+"+avx,+avx2"))
		variant.RemoveStringAttributeAtIndex(-1, EntryAttribute)
		variant.RemoveStringAttributeAtIndex(-1, InlineEntryAttribute)
		p.variants[fn] = variant
		if p.roots[fn] {
			baseline := fn.CloneFunction()
			baseline.SetName(fn.Name() + baselineSuffix)
			baseline.SetLinkage(llvm.InternalLinkage)
			baseline.RemoveStringAttributeAtIndex(-1, EntryAttribute)
			baseline.RemoveStringAttributeAtIndex(-1, InlineEntryAttribute)
			p.origins[baseline] = fn
			p.baselines[fn] = baseline
		}
	}
	for _, fn := range p.originals {
		if fn.IsDeclaration() {
			continue
		}
		variant := p.variants[fn]
		specialize(ctx, variant, p.variants, true)
		if baseline := p.baselines[fn]; !baseline.IsNil() {
			specialize(ctx, baseline, p.baselines, false)
		}
	}
}

func (p *avx2Plan) commit(mod llvm.Module, source *sourceInfo, implementations map[llvm.Value]dispatchPlan) {
	if p == nil {
		return
	}
	if source == nil {
		source = newSourceInfo(mod)
	}
	for _, fn := range p.originals {
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
			emitDispatch(mod, source, fn, dispatchPlan{baseline: baseline, choices: []dispatchChoice{{bits: featureAVX2, fn: variant}}}, implementations)
		}
	}
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

// A choice requires all bits and, if specified, at least one of anyOf.
type dispatchChoice struct {
	bits, anyOf uint64
	fn          llvm.Value
}

type dispatchPlan struct {
	baseline llvm.Value
	choices  []dispatchChoice
}

// A canonical AVX2 entry and its baseline may have additional target choices.
// Emit those choices directly into the public resolver so its slot always
// caches a physical body, including when the AVX2 entry is exported.
func (p dispatchPlan) selectBody(b llvm.Builder, features llvm.Value, implementations map[llvm.Value]dispatchPlan) llvm.Value {
	body := func(fn llvm.Value) llvm.Value {
		if sub, ok := implementations[fn]; ok {
			return sub.selectBody(b, features, nil)
		}
		return fn
	}
	selected := body(p.baseline)
	for _, choice := range p.choices {
		var condition llvm.Value
		if choice.bits != 0 {
			required := llvm.ConstInt(features.Type(), choice.bits, false)
			condition = b.CreateICmp(llvm.IntEQ, b.CreateAnd(features, required, ""), required, "")
		}
		if choice.anyOf != 0 {
			alternatives := llvm.ConstInt(features.Type(), choice.anyOf, false)
			present := b.CreateICmp(llvm.IntNE, b.CreateAnd(features, alternatives, ""), llvm.ConstNull(features.Type()), "")
			if condition.IsNil() {
				condition = present
			} else {
				condition = b.CreateAnd(condition, present, "")
			}
		}
		target := body(choice.fn)
		if condition.IsNil() {
			selected = target
		} else {
			selected = b.CreateSelect(condition, target, selected, "selected")
		}
	}
	return selected
}

func dispatch(mod llvm.Module, source *sourceInfo, fn, baseline llvm.Value, choices []dispatchChoice) {
	emitDispatch(mod, source, fn, dispatchPlan{baseline: baseline, choices: choices}, nil)
}

func emitDispatch(mod llvm.Module, source *sourceInfo, fn llvm.Value, plan dispatchPlan, implementations map[llvm.Value]dispatchPlan) {
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
	baseline := plan.baseline
	if sub, ok := implementations[baseline]; ok {
		baseline = sub.baseline
	}
	call := tailReturn(b, resolver, baseline)
	call.AddCallSiteAttribute(-1, ctx.CreateEnumAttribute(llvm.AttributeKindID("noinline"), 0))
	b.SetInsertPointAtEnd(selectImpl)
	selected := plan.selectBody(b, features, implementations)
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
	call.SetTailCallKind(x86ABIFor(fn.GlobalParent()).forwardingKind(fn))
	if fn.GlobalValueType().ReturnType().TypeKind() == llvm.VoidTypeKind {
		b.CreateRetVoid()
	} else {
		b.CreateRet(call)
	}
	return call
}
