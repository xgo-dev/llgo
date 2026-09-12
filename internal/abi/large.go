// Package abi contains target-independent lowering for LLGo's internal ABI.
package abi

import (
	"github.com/xgo-dev/llgo/internal/funcattrs"
	"github.com/xgo-dev/llvm"
)

const (
	// MaxStackVarSize matches cmd/compile's default limit for explicitly
	// declared variables. Values larger than this must not live on LLGo's
	// fixed native stack.
	MaxStackVarSize uint64 = 128 * 1024
	// MaxImplicitStackVarSize matches cmd/compile's default limit for
	// compiler-generated temporaries.
	MaxImplicitStackVarSize uint64 = 64 * 1024

	runtimeAllocU = "github.com/xgo-dev/llgo/runtime/internal/runtime.AllocU"
)

// LowerLargeAggregates converts oversized direct aggregate returns and copies
// to indirect memory operations before target-specific C ABI lowering runs.
func LowerLargeAggregates(td llvm.TargetData, m llvm.Module) {
	l := largeAggregateLowerer{td: td}
	l.transformModule(m)
}

type largeAggregateLowerer struct {
	td llvm.TargetData
}

func (l largeAggregateLowerer) isLargeAggregate(typ llvm.Type) bool {
	switch typ.TypeKind() {
	case llvm.ArrayTypeKind, llvm.StructTypeKind:
		return l.td.TypeAllocSize(typ) > MaxImplicitStackVarSize
	}
	return false
}

func (l largeAggregateLowerer) indirectType(ctx llvm.Context, typ llvm.Type) llvm.Type {
	params := append([]llvm.Type{llvm.PointerType(typ.ReturnType(), 0)}, typ.ParamTypes()...)
	return llvm.FunctionType(ctx.VoidType(), params, typ.IsFunctionVarArg())
}

func (l largeAggregateLowerer) transformModule(m llvm.Module) {
	var calls []llvm.Value
	var funcs []llvm.Value
	for fn := m.FirstFunction(); !fn.IsNil(); fn = llvm.NextFunction(fn) {
		if fn.IntrinsicID() == 0 && l.isLargeAggregate(fn.GlobalValueType().ReturnType()) {
			funcs = append(funcs, fn)
		}
		for bb := fn.FirstBasicBlock(); !bb.IsNil(); bb = llvm.NextBasicBlock(bb) {
			for instr := bb.FirstInstruction(); !instr.IsNil(); instr = llvm.NextInstruction(instr) {
				if !instr.IsAInvokeInst().IsNil() && l.isLargeAggregate(instr.CalledFunctionType().ReturnType()) {
					panic("large ABI: invoke requires unsupported indirect result conversion")
				}
				if call := instr.IsACallInst(); !call.IsNil() &&
					call.CalledValue().IntrinsicID() == 0 &&
					l.isLargeAggregate(call.CalledFunctionType().ReturnType()) {
					calls = append(calls, call)
				}
			}
		}
	}
	for _, call := range calls {
		l.transformCall(m, call)
	}
	for _, fn := range funcs {
		l.transformFunc(m, fn)
	}
	l.scalarizeExtractedLoads(m)
	l.transformStoredLoads(m)
}

// scalarizeExtractedLoads keeps projected fields from forcing an otherwise
// indirect large value back through SelectionDAG. This includes the leaf uses
// introduced by source value contracts. Read each projection at the original
// aggregate load: extracting it later must still observe that saved value if
// the source storage has since changed.
func (l largeAggregateLowerer) scalarizeExtractedLoads(m llvm.Module) {
	var loads []llvm.Value
	for fn := m.FirstFunction(); !fn.IsNil(); fn = llvm.NextFunction(fn) {
		for bb := fn.FirstBasicBlock(); !bb.IsNil(); bb = llvm.NextBasicBlock(bb) {
			for instr := bb.FirstInstruction(); !instr.IsNil(); instr = llvm.NextInstruction(instr) {
				load := instr.IsALoadInst()
				if !load.IsNil() && !load.IsVolatile() && l.isLargeAggregate(load.Type()) {
					loads = append(loads, load)
				}
			}
		}
	}
	for _, load := range loads {
		l.scalarizeLoadProjections(m.Context(), load)
		if load.FirstUse().IsNil() {
			load.EraseFromParentAsInstruction()
		}
	}
}

func (l largeAggregateLowerer) scalarizeLoadProjections(ctx llvm.Context, load llvm.Value) {
	b := ctx.NewBuilder()
	defer b.Dispose()
	b.SetInsertPointBefore(load)
	rootType := load.Type()
	rootAlign := load.Alignment()
	if rootAlign == 0 {
		rootAlign = l.td.ABITypeAlignment(rootType)
	}
	var visit func(llvm.Value, []uint32)
	visit = func(value llvm.Value, path []uint32) {
		var extracts []llvm.Value
		for use := value.FirstUse(); !use.IsNil(); use = use.NextUse() {
			if extract := use.User().IsAExtractValueInst(); !extract.IsNil() {
				extracts = append(extracts, extract)
			}
		}
		for _, extract := range extracts {
			indices := append(append([]uint32(nil), path...), extract.Indices()...)
			if l.isLargeAggregate(extract.Type()) {
				visit(extract, indices)
				if extract.FirstUse().IsNil() {
					extract.EraseFromParentAsInstruction()
				}
				continue
			}
			gepIndices := []llvm.Value{llvm.ConstInt(ctx.Int32Type(), 0, false)}
			typ, offset := rootType, uint64(0)
			for _, index := range indices {
				switch typ.TypeKind() {
				case llvm.StructTypeKind:
					gepIndices = append(gepIndices, llvm.ConstInt(ctx.Int32Type(), uint64(index), false))
					offset += l.td.ElementOffset(typ, int(index))
					typ = typ.StructElementTypes()[index]
				case llvm.ArrayTypeKind:
					// Array indices are signed GEP operands. Use the target pointer
					// width so a valid large 64-bit array index cannot become negative.
					gepIndices = append(gepIndices, llvm.ConstInt(ctx.IntType(l.td.PointerSize()*8), uint64(index), false))
					typ = typ.ElementType()
					offset += uint64(index) * l.td.TypeAllocSize(typ)
				}
			}
			ptr := b.CreateGEP(rootType, load.Operand(0), gepIndices, "aggregate.field.addr")
			field := b.CreateLoad(extract.Type(), ptr, "aggregate.field")
			alignment := rootAlign
			for offset%uint64(alignment) != 0 {
				alignment /= 2
			}
			field.SetAlignment(alignment)
			field.InstructionSetDebugLoc(load.InstructionDebugLoc())
			extract.ReplaceAllUsesWith(field)
			extract.EraseFromParentAsInstruction()
		}
	}
	visit(load, nil)
}

// transformStoredLoads prevents a large aggregate load from reaching
// SelectionDAG as one enormous SSA value. Lower an adjacent load/store pair
// directly to memmove. When the value is stored later or more than once,
// preserve Go assignment semantics by taking one snapshot at the original
// load and copying that snapshot to every destination at the original sites.
func (l largeAggregateLowerer) transformStoredLoads(m llvm.Module) {
	var loads []llvm.Value
	for fn := m.FirstFunction(); !fn.IsNil(); fn = llvm.NextFunction(fn) {
		for bb := fn.FirstBasicBlock(); !bb.IsNil(); bb = llvm.NextBasicBlock(bb) {
			for instr := bb.FirstInstruction(); !instr.IsNil(); instr = llvm.NextInstruction(instr) {
				load := instr.IsALoadInst()
				if !load.IsNil() && !load.IsVolatile() && l.isLargeAggregate(load.Type()) {
					if _, ok := storedLoadUsers(load); ok {
						loads = append(loads, load)
					}
				}
			}
		}
	}
	for _, load := range loads {
		l.transformStoredLoad(m, load)
	}
}

func storedLoadUsers(load llvm.Value) ([]llvm.Value, bool) {
	var stores []llvm.Value
	for use := load.FirstUse(); !use.IsNil(); use = use.NextUse() {
		store := use.User().IsAStoreInst()
		if store.IsNil() || store.IsVolatile() || store.Operand(0) != load {
			return nil, false
		}
		stores = append(stores, store)
	}
	return stores, len(stores) != 0
}

func (l largeAggregateLowerer) transformStoredLoad(m llvm.Module, load llvm.Value) {
	stores, ok := storedLoadUsers(load)
	if !ok {
		return
	}
	ctx := m.Context()
	b := ctx.NewBuilder()
	defer b.Dispose()
	typ := load.Type()

	b.SetInsertPointBefore(load)
	if len(stores) == 1 && llvm.NextInstruction(load) == stores[0] {
		copy := l.callMemmove(ctx, b, stores[0].Operand(1), load.Operand(0), typ)
		copy.InstructionSetDebugLoc(load.InstructionDebugLoc())
		stores[0].EraseFromParentAsInstruction()
		load.EraseFromParentAsInstruction()
		return
	}
	snapshot := l.allocResult(m, ctx, b, typ)
	copy := l.callMemcpy(ctx, b, snapshot, load.Operand(0), typ)
	copy.InstructionSetDebugLoc(load.InstructionDebugLoc())
	for _, store := range stores {
		b.SetInsertPointBefore(store)
		copy := l.callMemcpy(ctx, b, store.Operand(1), snapshot, typ)
		copy.InstructionSetDebugLoc(store.InstructionDebugLoc())
		store.EraseFromParentAsInstruction()
	}
	load.EraseFromParentAsInstruction()
}

func (l largeAggregateLowerer) transformCall(m llvm.Module, call llvm.Value) {
	ctx := m.Context()
	oldType := call.CalledFunctionType()
	retType := oldType.ReturnType()
	newType := l.indirectType(ctx, oldType)
	b := ctx.NewBuilder()
	defer b.Dispose()
	b.SetInsertPointBefore(call)

	result := l.allocResult(m, ctx, b, retType)
	params := make([]llvm.Value, 1, oldType.ParamTypesCount()+1)
	params[0] = result
	reflectMethodByName := call.GetCallSiteStringAttribute(-1, "llgo.reflect.methodbyname")
	reflectNameParam := -1
	for i := 0; i < oldType.ParamTypesCount(); i++ {
		params = append(params, call.Operand(i))
		if !call.GetCallSiteStringAttribute(i+1, "llgo.reflect.methodbyname.name").IsNil() {
			reflectNameParam = i + 2
		}
	}
	newCall := llvm.CreateCall(b, newType, call.CalledValue(), params)
	newCall.AddCallSiteAttribute(1, sretAttribute(ctx, retType))
	copyClosureEnvCallAttrs(call, newCall, 1)
	if err := funcattrs.RemapCall(call, newCall, indirectAttributeMapping(oldType.ParamTypesCount())); err != nil {
		panic(err)
	}
	if !reflectMethodByName.IsNil() {
		newCall.AddCallSiteAttribute(-1, reflectMethodByName)
	}
	if reflectNameParam >= 0 {
		newCall.AddCallSiteAttribute(reflectNameParam, ctx.CreateStringAttribute(
			"llgo.reflect.methodbyname.name", "1",
		))
	}
	newCall.SetInstructionCallConv(call.InstructionCallConv())
	newCall.InstructionSetDebugLoc(call.InstructionDebugLoc())

	value := b.CreateLoad(retType, result, "")
	value.InstructionSetDebugLoc(call.InstructionDebugLoc())
	call.ReplaceAllUsesWith(value)
	call.EraseFromParentAsInstruction()
	// Contract projections must not turn the fresh result transfer slot into
	// another heap snapshot. Read their leaves now, then retain the existing
	// direct copies from this private result allocation to its destinations.
	l.scalarizeLoadProjections(ctx, value)
	l.rewriteStoredResult(ctx, value, result, retType)
}

func (l largeAggregateLowerer) transformFunc(m llvm.Module, fn llvm.Value) {
	ctx := m.Context()
	oldType := fn.GlobalValueType()
	retType := oldType.ReturnType()
	newType := l.indirectType(ctx, oldType)
	name := fn.Name()
	fn.SetName("")
	nfn := llvm.AddFunction(m, name, newType)
	nfn.SetLinkage(fn.Linkage())
	nfn.SetFunctionCallConv(fn.FunctionCallConv())
	nfn.AddAttributeAtIndex(1, sretAttribute(ctx, retType))
	for _, attr := range fn.GetFunctionAttributes() {
		nfn.AddFunctionAttr(attr)
	}
	for i := 0; i < oldType.ParamTypesCount(); i++ {
		for _, attr := range fn.GetAttributesAtIndex(i + 1) {
			nfn.AddAttributeAtIndex(i+2, attr)
		}
	}
	if err := funcattrs.RemapFunction(fn, nfn, indirectAttributeMapping(oldType.ParamTypesCount())); err != nil {
		panic(err)
	}
	if sp := fn.Subprogram(); !sp.IsNil() {
		nfn.SetSubprogram(sp)
	}

	if !fn.IsDeclaration() {
		var blocks []llvm.BasicBlock
		for bb := fn.FirstBasicBlock(); !bb.IsNil(); bb = llvm.NextBasicBlock(bb) {
			blocks = append(blocks, bb)
		}
		for _, bb := range blocks {
			bb.RemoveFromParent()
			llvm.AppendExistingBasicBlock(nfn, bb)
		}
		for i := 0; i < oldType.ParamTypesCount(); i++ {
			fn.Param(i).ReplaceAllUsesWith(nfn.Param(i + 1))
		}
		l.rewriteReturns(ctx, nfn, retType)
	}

	fn.ReplaceAllUsesWith(nfn)
	fn.EraseFromParentAsFunction()
}

func (l largeAggregateLowerer) rewriteReturns(ctx llvm.Context, fn llvm.Value, retType llvm.Type) {
	var returns []llvm.Value
	var selfCopies [][2]llvm.Value
	for bb := fn.FirstBasicBlock(); !bb.IsNil(); bb = llvm.NextBasicBlock(bb) {
		for instr := bb.FirstInstruction(); !instr.IsNil(); instr = llvm.NextInstruction(instr) {
			if !instr.IsAReturnInst().IsNil() {
				returns = append(returns, instr)
				continue
			}
			store := instr.IsAStoreInst()
			if store.IsNil() || store.IsVolatile() {
				continue
			}
			load := store.Operand(0).IsALoadInst()
			if !load.IsNil() && !load.IsVolatile() && l.isLargeAggregate(load.Type()) &&
				store.Operand(1) == load.Operand(0) && hasSingleUse(load, store) {
				selfCopies = append(selfCopies, [2]llvm.Value{load, store})
			}
		}
	}
	for _, copy := range selfCopies {
		copy[1].EraseFromParentAsInstruction()
		copy[0].EraseFromParentAsInstruction()
	}

	b := ctx.NewBuilder()
	defer b.Dispose()
	result := fn.Param(0)
	for _, ret := range returns {
		value := ret.Operand(0)
		load := value.IsALoadInst()
		if !load.IsNil() && !load.IsVolatile() && hasSingleUse(load, ret) {
			// Copy at the original load so a later source mutation cannot change
			// the already-evaluated return value (issue #1608).
			b.SetInsertPointBefore(load)
			l.callMemcpy(ctx, b, result, load.Operand(0), retType)
			b.SetInsertPointBefore(ret)
			b.CreateRetVoid()
			ret.EraseFromParentAsInstruction()
			load.EraseFromParentAsInstruction()
			continue
		}
		b.SetInsertPointBefore(ret)
		b.CreateStore(value, result)
		b.CreateRetVoid()
		ret.EraseFromParentAsInstruction()
	}
}

func (l largeAggregateLowerer) rewriteStoredResult(ctx llvm.Context, value, result llvm.Value, typ llvm.Type) {
	var stores []llvm.Value
	for use := value.FirstUse(); !use.IsNil(); use = use.NextUse() {
		store := use.User().IsAStoreInst()
		if store.IsNil() || store.IsVolatile() || store.Operand(0) != value {
			return
		}
		stores = append(stores, store)
	}
	b := ctx.NewBuilder()
	defer b.Dispose()
	for _, store := range stores {
		b.SetInsertPointBefore(store)
		l.callMemcpy(ctx, b, store.Operand(1), result, typ)
		store.EraseFromParentAsInstruction()
	}
	if len(stores) != 0 || value.FirstUse().IsNil() {
		value.EraseFromParentAsInstruction()
	}
}

func (l largeAggregateLowerer) allocResult(m llvm.Module, ctx llvm.Context, b llvm.Builder, typ llvm.Type) llvm.Value {
	if err := funcattrs.CheckInstrumentation(b.GetInsertBlock().Parent(), "large ABI result allocation",
		"memory", "nofree", "nosync", "nounwind", "willreturn", "capture", "access", "noalias"); err != nil {
		panic(err)
	}
	intType := ctx.IntType(l.td.PointerSize() * 8)
	ptrType := llvm.PointerType(ctx.Int8Type(), 0)
	fnType := llvm.FunctionType(ptrType, []llvm.Type{intType}, false)
	fn := m.NamedFunction(runtimeAllocU)
	if fn.IsNil() {
		fn = llvm.AddFunction(m, runtimeAllocU, fnType)
	}
	size := llvm.ConstInt(intType, l.td.TypeAllocSize(typ), false)
	return llvm.CreateCall(b, fnType, fn, []llvm.Value{size})
}

func (l largeAggregateLowerer) callMemcpy(ctx llvm.Context, b llvm.Builder, dst, src llvm.Value, typ llvm.Type) llvm.Value {
	return l.callMemoryCopy(ctx, b, "llvm.memcpy", dst, src, typ)
}

func (l largeAggregateLowerer) callMemmove(ctx llvm.Context, b llvm.Builder, dst, src llvm.Value, typ llvm.Type) llvm.Value {
	return l.callMemoryCopy(ctx, b, "llvm.memmove", dst, src, typ)
}

func (l largeAggregateLowerer) callMemoryCopy(ctx llvm.Context, b llvm.Builder, intrinsic string, dst, src llvm.Value, typ llvm.Type) llvm.Value {
	size := llvm.ConstInt(ctx.IntType(l.td.PointerSize()*8), l.td.TypeAllocSize(typ), false)
	return b.CreateIntrinsic(ctx.VoidType(), llvm.LookupIntrinsicID(intrinsic), []llvm.Value{
		dst, src, size, llvm.ConstInt(ctx.Int1Type(), 0, false),
	}, "")
}

func sretAttribute(ctx llvm.Context, typ llvm.Type) llvm.Attribute {
	return ctx.CreateTypeAttribute(llvm.AttributeKindID("sret"), typ)
}

func copyClosureEnvCallAttrs(from, to llvm.Value, paramOffset int) {
	for i := 0; i < from.CalledFunctionType().ParamTypesCount(); i++ {
		for _, name := range []string{"nest", "swiftself"} {
			kind := llvm.AttributeKindID(name)
			if attr := from.GetCallSiteEnumAttribute(i+1, kind); !attr.IsNil() {
				to.AddCallSiteAttribute(i+1+paramOffset, attr)
			}
		}
	}
}

func hasSingleUse(value, user llvm.Value) bool {
	use := value.FirstUse()
	return !use.IsNil() && use.User() == user && use.NextUse().IsNil()
}
