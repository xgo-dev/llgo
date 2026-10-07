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
	variantSuffix  = ".__llgo_fmv_avx2"
	queryAttribute = "llgo.cpu.query"
	avx2Query      = "x86.avx2"
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
// SIMD128 functions. It must run before ABI lowering and target optimization,
// including in O0 and ModeGen builds. Running it again is harmless.
func Run(mod llvm.Module) error {
	arch, _, _ := strings.Cut(mod.Target(), "-")
	if arch != "x86_64" && arch != "amd64" && arch != "x86_64h" {
		return nil
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
		name := fn.Name() + variantSuffix
		// A conflicting alias/global is just as invalid as a function collision.
		if !mod.NamedValue(name).IsNil() {
			return fmt.Errorf("SIMD FMV symbol collision: %s", name)
		}
		originals = append(originals, fn)
	}
	if len(originals) == 0 {
		return nil
	}

	ctx := mod.Context()
	source := newSourceInfo(mod)
	variants := make(map[llvm.Value]llvm.Value, len(originals))
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
		variant.AddFunctionAttr(ctx.CreateStringAttribute(doneAttribute, ""))
		fn.AddFunctionAttr(ctx.CreateStringAttribute(doneAttribute, ""))
		variants[fn] = variant
	}
	for _, fn := range originals {
		if fn.IsDeclaration() {
			continue
		}
		variant := variants[fn]
		specialize(ctx, variant, variants)
		if query := roots[fn]; !query.IsNil() {
			dispatch(ctx, fn, variant, query)
		}
	}
	return nil
}

func specialize(ctx llvm.Context, fn llvm.Value, variants map[llvm.Value]llvm.Value) {
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
		query.ReplaceAllUsesWith(llvm.ConstInt(ctx.Int1Type(), 1, false))
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

func dispatch(ctx llvm.Context, fn, variant, query llvm.Value) {
	baseline := fn.EntryBasicBlock()
	entry := ctx.InsertBasicBlock(baseline, "fmv.entry")
	fast := ctx.InsertBasicBlock(baseline, "fmv.avx2")
	b := ctx.NewBuilder()
	defer b.Dispose()
	b.SetInsertPointAtEnd(entry)
	if sp := fn.Subprogram(); !sp.IsNil() {
		b.SetCurrentDebugLocation(0, 0, sp, llvm.Metadata{})
	}
	var queryArgs []llvm.Value
	for _, param := range query.Params() {
		queryArgs = append(queryArgs, llvm.ConstNull(param.Type()))
	}
	check := b.CreateCall(query.GlobalValueType(), query, queryArgs, "")
	check.SetInstructionCallConv(query.FunctionCallConv())
	b.CreateCondBr(check, fast, baseline)
	b.SetInsertPointAtEnd(fast)
	call := b.CreateCall(variant.GlobalValueType(), variant, fn.Params(), "")
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
}
