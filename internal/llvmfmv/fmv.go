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
	InlineEntryAttribute = attributePrefix + "inline-entry"
	QueryAttribute       = "llgo.cpu.query"
	PreferFeatureMD      = attributePrefix + "prefer-feature"
	AVX2Query            = "x86.avx2"

	moduleDone     = "llgo.fmv.done"
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
// and specializes local SIMD work. It runs after ABI
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
	switch baseline {
	case "", "v1", "v2", "v3", "v4":
	default:
		return fmt.Errorf("SIMD FMV invalid GOAMD64 baseline: %s", baseline)
	}
	if err := planVersions(abi, baseline); err != nil {
		return err
	}
	mod.AddNamedMetadataOperand(moduleDone, mod.Context().MDNode(nil))
	return nil
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

func (p dispatchPlan) selectBody(b llvm.Builder, features llvm.Value) llvm.Value {
	selected := p.baseline
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
		target := choice.fn
		if condition.IsNil() {
			selected = target
		} else {
			selected = b.CreateSelect(condition, target, selected, "selected")
		}
	}
	return selected
}

func emitDispatch(mod llvm.Module, source *sourceInfo, fn llvm.Value, plan dispatchPlan) {
	ctx := mod.Context()
	resolver := fn.CloneFunction()
	resolver.SetName(fn.Name() + resolverSuffix)
	resolver.SetLinkage(llvm.InternalLinkage)
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
	call := tailReturn(b, resolver, plan.baseline)
	call.AddCallSiteAttribute(-1, ctx.CreateEnumAttribute(llvm.AttributeKindID("noinline"), 0))
	b.SetInsertPointAtEnd(selectImpl)
	selected := plan.selectBody(b, features)
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
