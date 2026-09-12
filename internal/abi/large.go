// Package abi contains target-independent lowering for LLGo's internal ABI.
package abi

import "github.com/xgo-dev/llvm"

const (
	// MaxStackVarSize matches cmd/compile's default limit for explicitly
	// declared variables. Values larger than this must not live on LLGo's
	// fixed native stack.
	MaxStackVarSize uint64 = 128 * 1024
	// MaxImplicitStackVarSize matches cmd/compile's default limit for
	// compiler-generated temporaries.
	MaxImplicitStackVarSize uint64 = 64 * 1024
	// MinWasmAggregateCopySize is the point at which Wasm aggregate loads and
	// stores are lowered to memory intrinsics to avoid LLVM scalarization.
	MinWasmAggregateCopySize uint64 = 4 * 1024

	runtimeAllocU = "github.com/xgo-dev/llgo/runtime/internal/runtime.AllocU"
)

// AggregateLoweringConfig describes the Go runtime ABI used by lowering-created
// allocations and roots. GoWordSize can exceed the physical Wasm address size.
type AggregateLoweringConfig struct {
	GoWordSize int
	GCRoots    bool
	Wasm       bool
}

// LowerLargeAggregates converts oversized direct aggregate returns and copies
// to indirect memory operations before target-specific C ABI lowering runs.
func LowerLargeAggregates(td llvm.TargetData, m llvm.Module, config AggregateLoweringConfig) {
	l := newLargeAggregateLowerer(td, config)
	l.transformModule(m)
}

// LowerWasmAggregateCopies applies the same snapshot lowering to copies of at
// least 4 KiB. LLVM scalarizes these too, notably in reflection's by-value
// wrappers. Return types and the native stack/return ABI limits are unchanged.
func LowerWasmAggregateCopies(td llvm.TargetData, m llvm.Module, config AggregateLoweringConfig) int {
	l := newLargeAggregateLowerer(td, config)
	l.copyMinSize = MinWasmAggregateCopySize
	changed := 0
	for {
		count := l.transformStoredLoads(m)
		if count == 0 {
			break
		}
		changed += count
		// Projecting a field of an aggregate snapshot can expose another
		// large load, for example the array argument inside a deferred call's
		// closure. Lower those loads too before handing the module to LLVM.
		// Each projection descends into a strictly nested aggregate type.
	}
	if l.roots {
		l.publishRoots(m)
	}
	return changed
}

type aggregateRoot struct {
	value, before llvm.Value
}

type largeAggregateLowerer struct {
	td           llvm.TargetData
	goWordSize   int
	roots        bool
	wasm         bool
	copyMinSize  uint64
	allocations  []llvm.Value
	resultParams []llvm.Value
	sourceRoots  []aggregateRoot
}

func newLargeAggregateLowerer(td llvm.TargetData, config AggregateLoweringConfig) largeAggregateLowerer {
	goWordSize := config.GoWordSize
	if goWordSize == 0 {
		goWordSize = td.PointerSize()
	}
	return largeAggregateLowerer{
		td:         td,
		goWordSize: goWordSize,
		roots:      config.GCRoots,
		wasm:       config.Wasm,
	}
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

func (l largeAggregateLowerer) isLargeCopy(typ llvm.Type) bool {
	if l.copyMinSize == 0 {
		return l.isLargeAggregate(typ)
	}
	switch typ.TypeKind() {
	case llvm.ArrayTypeKind, llvm.StructTypeKind:
		return l.td.TypeAllocSize(typ) >= l.copyMinSize
	}
	return false
}

func (l *largeAggregateLowerer) transformModule(m llvm.Module) {
	var calls []llvm.Value
	var funcs []llvm.Value
	for fn := m.FirstFunction(); !fn.IsNil(); fn = llvm.NextFunction(fn) {
		if fn.IntrinsicID() == 0 && l.isLargeAggregate(fn.GlobalValueType().ReturnType()) {
			funcs = append(funcs, fn)
		}
		for bb := fn.FirstBasicBlock(); !bb.IsNil(); bb = llvm.NextBasicBlock(bb) {
			for instr := bb.FirstInstruction(); !instr.IsNil(); instr = llvm.NextInstruction(instr) {
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
	l.transformStoredLoads(m)
	if l.roots {
		l.publishRoots(m)
	}
}

// transformStoredLoads prevents a large aggregate load from reaching
// SelectionDAG as one enormous SSA value. Lower an adjacent load/store pair
// directly to memmove. When the value is stored later or more than once,
// preserve Go assignment semantics by taking one snapshot at the original
// load and copying that snapshot to every destination at the original sites.
func (l *largeAggregateLowerer) transformStoredLoads(m llvm.Module) int {
	var loads []llvm.Value
	var zeroStores []llvm.Value
	for fn := m.FirstFunction(); !fn.IsNil(); fn = llvm.NextFunction(fn) {
		for bb := fn.FirstBasicBlock(); !bb.IsNil(); bb = llvm.NextBasicBlock(bb) {
			for instr := bb.FirstInstruction(); !instr.IsNil(); instr = llvm.NextInstruction(instr) {
				if store := instr.IsAStoreInst(); !store.IsNil() && store.Operand(0).IsNull() && l.isLargeCopy(store.Operand(0).Type()) {
					zeroStores = append(zeroStores, store)
				}
				load := instr.IsALoadInst()
				if !load.IsNil() && l.isLargeCopy(load.Type()) {
					if _, _, ok := l.aggregateUsers(load); ok {
						loads = append(loads, load)
					}
				}
			}
		}
	}
	for _, load := range loads {
		l.transformStoredLoad(m, load)
	}
	// A deferred result starts with a volatile zero store. Keep the barrier
	// while avoiding a SelectionDAG value with one operand per byte.
	ctx := m.Context()
	b := ctx.NewBuilder()
	defer b.Dispose()
	for _, store := range zeroStores {
		b.SetInsertPointBefore(store)
		size := llvm.ConstInt(ctx.IntType(l.td.PointerSize()*8), l.td.TypeAllocSize(store.Operand(0).Type()), false)
		zero := b.CreateIntrinsic(ctx.VoidType(), llvm.LookupIntrinsicID("llvm.memset"), []llvm.Value{
			store.Operand(1), llvm.ConstInt(ctx.Int8Type(), 0, false), size, llvm.ConstInt(ctx.Int1Type(), 0, false),
		}, "")
		setCopyVolatile(ctx, zero, store.IsVolatile())
		zero.InstructionSetDebugLoc(store.InstructionDebugLoc())
		store.EraseFromParentAsInstruction()
	}
	return len(loads) + len(zeroStores)
}

func (l largeAggregateLowerer) aggregateUsers(value llvm.Value) (stores, extracts []llvm.Value, ok bool) {
	for use := value.FirstUse(); !use.IsNil(); use = use.NextUse() {
		user := use.User()
		if store := user.IsAStoreInst(); !store.IsNil() && store.Operand(0) == value {
			stores = append(stores, store)
		} else if extract := user.IsAExtractValueInst(); !extract.IsNil() && !l.isLargeAggregate(extract.Type()) {
			extracts = append(extracts, extract)
		} else {
			return nil, nil, false
		}
	}
	return stores, extracts, len(stores)+len(extracts) != 0
}

func (l *largeAggregateLowerer) transformStoredLoad(m llvm.Module, load llvm.Value) {
	stores, extracts, ok := l.aggregateUsers(load)
	if !ok {
		return
	}
	ctx := m.Context()
	b := ctx.NewBuilder()
	defer b.Dispose()
	typ := load.Type()

	b.SetInsertPointBefore(load)
	if len(stores) == 1 && len(extracts) == 0 && llvm.NextInstruction(load) == stores[0] {
		copy := l.callMemmove(ctx, b, stores[0].Operand(1), load.Operand(0), typ)
		setCopyVolatile(ctx, copy, load.IsVolatile() || stores[0].IsVolatile())
		copy.InstructionSetDebugLoc(load.InstructionDebugLoc())
		stores[0].EraseFromParentAsInstruction()
		load.EraseFromParentAsInstruction()
		return
	}
	snapshot := l.allocResult(m, ctx, b, typ)
	// This allocation is a new safepoint that was absent from the frontend's
	// root plan. Keep the source alive before allocating, not only the result
	// afterwards; reflection wrappers can have no original allocation at all.
	l.sourceRoots = append(l.sourceRoots, aggregateRoot{value: load.Operand(0), before: snapshot})
	copy := l.callMemcpy(ctx, b, snapshot, load.Operand(0), typ)
	setCopyVolatile(ctx, copy, load.IsVolatile())
	copy.InstructionSetDebugLoc(load.InstructionDebugLoc())
	l.rewriteMemoryUsers(ctx, load, snapshot, typ, stores, extracts)
}

func (l *largeAggregateLowerer) transformCall(m llvm.Module, call llvm.Value) {
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
	l.rewriteStoredResult(ctx, value, result, retType)
}

func (l *largeAggregateLowerer) transformFunc(m llvm.Module, fn llvm.Value) {
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
		l.resultParams = append(l.resultParams, nfn.Param(0))
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
	stores, extracts, ok := l.aggregateUsers(value)
	if !ok && !value.FirstUse().IsNil() {
		return
	}
	l.rewriteMemoryUsers(ctx, value, result, typ, stores, extracts)
}

// Root publication extracts pointer members from aggregate SSA values. Rewrite
// those projections as loads from the same immutable snapshot as its stores;
// leaving even one aggregate use expands the entire value in SelectionDAG.
func (l largeAggregateLowerer) rewriteMemoryUsers(ctx llvm.Context, value, result llvm.Value, typ llvm.Type, stores, extracts []llvm.Value) {
	b := ctx.NewBuilder()
	defer b.Dispose()
	for _, store := range stores {
		b.SetInsertPointBefore(store)
		copy := l.callMemcpy(ctx, b, store.Operand(1), result, typ)
		setCopyVolatile(ctx, copy, store.IsVolatile())
		copy.InstructionSetDebugLoc(store.InstructionDebugLoc())
		store.EraseFromParentAsInstruction()
	}
	for _, extract := range extracts {
		b.SetInsertPointBefore(extract)
		indices := []llvm.Value{llvm.ConstInt(ctx.Int32Type(), 0, false)}
		for _, index := range extract.Indices() {
			indices = append(indices, llvm.ConstInt(ctx.Int32Type(), uint64(index), false))
		}
		field := b.CreateInBoundsGEP(typ, result, indices, "")
		projected := b.CreateLoad(extract.Type(), field, "")
		projected.InstructionSetDebugLoc(extract.InstructionDebugLoc())
		extract.ReplaceAllUsesWith(projected)
		extract.EraseFromParentAsInstruction()
	}
	value.EraseFromParentAsInstruction()
}

func setCopyVolatile(ctx llvm.Context, copy llvm.Value, volatile bool) {
	if volatile {
		copy.SetOperand(3, llvm.ConstInt(ctx.Int1Type(), 1, false))
	}
}

func (l *largeAggregateLowerer) allocResult(m llvm.Module, ctx llvm.Context, b llvm.Builder, typ llvm.Type) llvm.Value {
	intType := ctx.IntType(l.goWordSize * 8)
	ptrType := llvm.PointerType(ctx.Int8Type(), 0)
	fnType := llvm.FunctionType(ptrType, []llvm.Type{intType}, false)
	fn := m.NamedFunction(runtimeAllocU)
	if fn.IsNil() {
		fn = llvm.AddFunction(m, runtimeAllocU, fnType)
	}
	size := llvm.ConstInt(intType, l.td.TypeAllocSize(typ), false)
	result := llvm.CreateCall(b, fnType, fn, []llvm.Value{size})
	l.allocations = append(l.allocations, result)
	return result
}

func (l *largeAggregateLowerer) publishRoots(m llvm.Module) {
	byFunc := make(map[llvm.Value][]aggregateRoot)
	for _, value := range l.allocations {
		fn := value.InstructionParent().Parent()
		byFunc[fn] = append(byFunc[fn], aggregateRoot{value: value, before: llvm.NextInstruction(value)})
	}
	for _, value := range l.resultParams {
		fn := value.ParamParent()
		byFunc[fn] = append(byFunc[fn], aggregateRoot{value: value, before: fn.FirstBasicBlock().FirstInstruction()})
	}
	for _, root := range l.sourceRoots {
		fn := root.before.InstructionParent().Parent()
		byFunc[fn] = append(byFunc[fn], root)
	}
	b := m.Context().NewBuilder()
	defer b.Dispose()
	for fn := m.FirstFunction(); !fn.IsNil(); fn = llvm.NextFunction(fn) {
		values := byFunc[fn]
		if len(values) == 0 {
			continue
		}
		frame := NewGCRootFrame(m, fn, len(values), l.td.PointerSize(), l.goWordSize, l.wasm)
		for i, root := range values {
			b.SetInsertPointBefore(root.before)
			frame.StorePointer(b, frame.Slots[i], root.value)
		}
		PopGCRootFrame(m, fn, frame)
	}
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
