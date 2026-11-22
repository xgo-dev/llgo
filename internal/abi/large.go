// Package abi contains target-independent lowering for LLGo's internal ABI.
package abi

import (
	"slices"

	"github.com/xgo-dev/llgo/internal/env"
	"github.com/xgo-dev/llgo/internal/llvmattr"
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
	// MinAggregateCopySize is the point at which aggregate loads and
	// stores are lowered to memory intrinsics to avoid LLVM scalarization.
	// Return types and the native stack/return ABI still use MaxImplicitStackVarSize.
	MinAggregateCopySize uint64 = 4 * 1024
	// MaxSSAValueWords matches cmd/compile's ssa.MaxStruct: a value may occupy
	// at most this many pointer-sized words and still be SSAable.
	MaxSSAValueWords = 4
	// MaxNativeRecoverStackValueSize is the upper bound for promoting a
	// recover-visible local aggregate to entry-frame stack storage on native
	// amd64 and arm64 targets. Promoted recover slots require a volatile zero
	// initialization and volatile owner accesses across SJLJ; up to 128 bytes
	// covers bounded structs and small buffers measured in #3208, while larger
	// captures remain on the heap because volatile zeroing, repeated volatile
	// loads/stores, and stack roots can cost more than a heap allocation.
	MaxNativeRecoverStackValueSize uint64 = 128

	// HeaderSnapshotMD marks a reload after the frontend has copied a value
	// and cleared its Wasm header padding in private storage.
	HeaderSnapshotMD = "llgo.header.snapshot"

	runtimeAllocU        = env.LLGoRuntimePkg + "/internal/runtime.AllocU"
	runtimeAllocUAligned = env.LLGoRuntimePkg + "/internal/runtime.AllocUAligned"
)

// MaxSSAValueSize is cmd/compile's CanSSA size limit (MaxStruct * PtrSize).
func MaxSSAValueSize(ptrSize int) uint64 {
	return uint64(MaxSSAValueWords * ptrSize)
}

// MaxRecoverStackValueSize returns the maximum size of a recover-visible local
// capture that may be promoted to entry-frame stack storage on goarch.
func MaxRecoverStackValueSize(goarch string, ptrSize int) uint64 {
	switch goarch {
	case "arm64", "amd64":
		return MaxNativeRecoverStackValueSize
	default:
		return MaxSSAValueSize(ptrSize)
	}
}

// ShouldLowerArrayCopy reports whether a multi-element array should be copied
// in memory instead of as a first-class LLVM value. Arrays larger than
// MaxSSAValueSize are not SSAable in cmd/compile; smaller ones stay in registers.
func ShouldLowerArrayCopy(length int, size uint64, ptrSize int) bool {
	return length > 1 && size > MaxSSAValueSize(ptrSize)
}

// ShouldSnapshotAggregateLoad reports whether ABI copy lowering may heap-allocate
// a snapshot of this load (AllocU), which the frontend must treat as a GC
// safepoint. Copies smaller than MinAggregateCopySize use a stack alloca
// instead, including CanSSA-sized arrays, so they are not safepoints.
func ShouldSnapshotAggregateLoad(isArray bool, arrayLen int, isStruct bool, size uint64, ptrSize int) bool {
	if (isArray || isStruct) && size >= MinAggregateCopySize {
		return true
	}
	return false
}

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

// LowerAggregateCopies applies snapshot lowering to copies of at least 4 KiB
// and to multi-element arrays larger than MaxSSAValueSize. LLVM scalarizes
// first-class loads of those arrays, notably NTT polynomials. Smaller arrays
// stay first-class so they can remain in registers. Function signatures, return
// sret, and the native C ABI are unchanged: loads used as call arguments are
// left alone. config.Wasm still selects the wasm GC-root frame layout.
func LowerAggregateCopies(td llvm.TargetData, m llvm.Module, config AggregateLoweringConfig) int {
	l := newLargeAggregateLowerer(td, config)
	l.copyMinSize = MinAggregateCopySize
	l.copyMultiElementArrays = true
	changed := 0
	// The pass is monotonic: every rewrite removes one qualifying aggregate
	// load, and can expose only projections into a strictly nested aggregate.
	// It never recreates the load or a containing projection it just removed.
	// The number of rounds follows the input's nesting depth, not a fixed cap.
	for {
		count := l.transformStoredLoads(m)
		if count == 0 {
			break
		}
		changed += count
		// Projecting a field of an aggregate snapshot can expose another
		// large load, for example the array argument inside a deferred call's
		// closure. Lower those loads too before handing the module to LLVM.
	}
	if l.roots {
		l.publishRoots(m)
	}
	return changed
}

type aggregateRoot struct {
	value, before llvm.Value
}

// snapshotRange is the live interval of one occupant of a stack snapshot slot:
// from the snapshot copy through every read of that copy.
type snapshotRange struct {
	start llvm.Value
	uses  []llvm.Value
	live  map[llvm.Value]struct{}
}

// stackSnapshotSlot is a loop-safe entry alloca that disjoint snapshots of the
// same type can share. Mutually exclusive branches therefore reserve peak
// simultaneous size rather than the sum of every snapshot.
type stackSnapshotSlot struct {
	ptr    llvm.Value
	typ    llvm.Type
	fn     llvm.Value
	ranges []snapshotRange
}

type debugLocation struct {
	loc       llvm.Metadata
	synthetic bool
	line      uint
	scope     llvm.Metadata
}

type largeAggregateLowerer struct {
	td                     llvm.TargetData
	goWordSize             int
	roots                  bool
	wasm                   bool
	copyMinSize            uint64
	copyMultiElementArrays bool
	allocations            []llvm.Value
	resultParams           []llvm.Value
	sourceRoots            []aggregateRoot
	stackSlots             []stackSnapshotSlot
	cfgFn                  llvm.Value
	bbIndex                map[llvm.BasicBlock]int
	bbs                    []llvm.BasicBlock
	bbPreds                [][]int
	debugLocs              map[llvm.Value]debugLocation
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

func (l largeAggregateLowerer) isArrayCopyToLower(typ llvm.Type) bool {
	if typ.TypeKind() != llvm.ArrayTypeKind {
		return false
	}
	return ShouldLowerArrayCopy(typ.ArrayLength(), l.td.TypeAllocSize(typ), l.td.PointerSize())
}

func (l largeAggregateLowerer) isLargeCopy(typ llvm.Type) bool {
	if l.copyMultiElementArrays && l.isArrayCopyToLower(typ) {
		return true
	}
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

// ABI lowering exposes call arguments and results as stores. Retry snapshot
// reuse before each copy pass, including the pass after C ABI lowering.
func (l *largeAggregateLowerer) reuseHeaderSnapshots(m llvm.Module) int {
	if !l.wasm {
		return 0
	}
	var loads []llvm.Value
	for fn := m.FirstFunction(); !fn.IsNil(); fn = llvm.NextFunction(fn) {
		for bb := fn.FirstBasicBlock(); !bb.IsNil(); bb = llvm.NextBasicBlock(bb) {
			for inst := bb.FirstInstruction(); !inst.IsNil(); inst = llvm.NextInstruction(inst) {
				if !inst.IsALoadInst().IsNil() && isHeaderSnapshot(inst) {
					loads = append(loads, inst)
				}
			}
		}
	}
	changed := 0
	for _, load := range loads {
		if l.reuseHeaderSnapshot(m.Context(), load) {
			changed++
		}
	}
	return changed
}

func isHeaderSnapshot(load llvm.Value) bool {
	return load.HasMetadata() && !load.Metadata(load.Type().Context().MDKindID(HeaderSnapshotMD)).IsNil()
}

// Heap snapshots are private to this evaluation. A stack slot is also usable
// when it has only one fill/clear/reload; shared frontend scratch can be
// overwritten by another load and must retain ordinary snapshot lowering.
func (l *largeAggregateLowerer) reuseHeaderSnapshot(ctx llvm.Context, load llvm.Value) bool {
	slot := load.Operand(0)
	heap := !slot.IsACallInst().IsNil()
	if heap {
		if name := slot.CalledValue().Name(); name != runtimeAllocU && name != runtimeAllocUAligned {
			return false
		}
	} else if slot.IsAAllocaInst().IsNil() || !isPrivateHeaderSnapshot(load) {
		return false
	}
	stores, extracts, ok := l.aggregateUsers(load, true)
	if !ok {
		return false
	}
	if len(stores) == 1 && len(extracts) == 0 && !stores[0].IsVolatile() &&
		l.forwardHeaderSnapshot(ctx, load, stores[0], slot) {
		return true
	}
	if heap {
		l.allocations = append(l.allocations, slot)
	}
	l.rewriteMemoryUsers(ctx, load, slot, l.td.ABITypeAlignment(load.Type()), stores, extracts)
	return true
}

func isPrivateHeaderSnapshot(load llvm.Value) bool {
	// The caller identifies the header clear. Its first argument and the
	// adjacent fill and reload must refer to the same private storage.
	slot := load.Operand(0)
	clear := llvm.PrevInstruction(load)
	if clear.IsNil() || clear.IsACallInst().IsNil() || clear.Operand(0) != slot {
		return false
	}
	fill := llvm.PrevInstruction(clear)
	if fill.IsNil() {
		return false
	}
	if !fill.IsAStoreInst().IsNil() {
		if fill.Operand(1) != slot {
			return false
		}
	} else if !isMemoryCopy(fill) || fill.Operand(0) != slot {
		return false
	}
	for use := slot.FirstUse(); !use.IsNil(); use = use.NextUse() {
		switch use.User() {
		case fill, clear, load:
		default:
			return false
		}
	}
	return true
}

func isMemoryCopy(inst llvm.Value) bool {
	if inst.IsACallInst().IsNil() {
		return false
	}
	id := inst.CalledValue().IntrinsicID()
	return id == llvm.LookupIntrinsicID("llvm.memcpy") || id == llvm.LookupIntrinsicID("llvm.memmove")
}

// Copy directly at the final store when only pure instructions and branches
// separate it from the snapshot. In particular, keep destination nil checks
// before both the copy and padding clear. Volatile reads retain their snapshot.
func (l *largeAggregateLowerer) forwardHeaderSnapshot(ctx llvm.Context, load, store, slot llvm.Value) bool {
	if !isPrivateHeaderSnapshot(load) {
		return false
	}
	clear := llvm.PrevInstruction(load)
	fill := llvm.PrevInstruction(clear)
	var src, read llvm.Value
	if !fill.IsAStoreInst().IsNil() {
		read = fill.Operand(0)
		if read.IsALoadInst().IsNil() || read.IsVolatile() || llvm.NextInstruction(read) != fill || !hasSingleUse(read, fill) {
			return false
		}
		src = read.Operand(0)
	} else {
		// A previous copy pass may already have lowered the snapshot fill.
		if !fill.Operand(3).IsNull() || fill.Operand(2).IsAConstantInt().IsNil() ||
			fill.Operand(2).ZExtValue() != l.td.TypeAllocSize(load.Type()) {
			return false
		}
		src = fill.Operand(1)
	}
	if !l.canSinkHeaderSnapshot(load, store) {
		return false
	}
	dst := store.Operand(1)
	b := ctx.NewBuilder()
	defer b.Dispose()
	b.SetInsertPointBefore(store)
	// The final destination can overlap the source even when the private
	// snapshot fill used memcpy. Do not retain its non-aliasing assumptions.
	copy := l.callMemmove(ctx, b, dst, src, load.Type())
	copy.InstructionSetDebugLoc(fill.InstructionDebugLoc())
	l.replaceTrackedUse(fill, copy)
	clear.SetOperand(0, dst)
	clear.RemoveFromParentAsInstruction()
	b.Insert(clear)
	fill.EraseFromParentAsInstruction()
	if !read.IsNil() {
		l.replaceTrackedUse(read, copy)
		read.EraseFromParentAsInstruction()
	}
	l.replaceTrackedUse(store, copy)
	store.EraseFromParentAsInstruction()
	load.EraseFromParentAsInstruction()
	slot.EraseFromParentAsInstruction()
	return true
}

// A single predecessor chain excludes joins and repeated stores without a
// fresh load. Calls on side exits, such as a failed nil check, do not interfere.
func (l *largeAggregateLowerer) canSinkHeaderSnapshot(load, store llvm.Value) bool {
	l.ensureCFG(load.InstructionParent().Parent())
	block := store.InstructionParent()
	seen := map[llvm.BasicBlock]bool{block: true}
	for inst := llvm.PrevInstruction(store); ; inst = llvm.PrevInstruction(inst) {
		if inst == load {
			return true
		}
		if inst.IsNil() {
			preds := l.bbPreds[l.bbIndex[block]]
			if len(preds) != 1 || seen[l.bbs[preds[0]]] {
				return false
			}
			block = l.bbs[preds[0]]
			seen[block] = true
			inst = block.LastInstruction()
		}
		// Reject all memory accesses, calls and synchronization, including
		// paths that could change the source or execute this store twice.
		switch inst.InstructionOpcode() {
		case llvm.GetElementPtr, llvm.ICmp, llvm.FCmp, llvm.PHI, llvm.Select, llvm.Br:
		default:
			if inst.IsABinaryOperator().IsNil() && inst.IsACastInst().IsNil() {
				return false
			}
		}
	}
}

// transformStoredLoads prevents a large aggregate load from reaching
// SelectionDAG as one enormous SSA value. Lower an adjacent load/store pair
// directly to memmove. When the value is stored later or more than once,
// preserve Go assignment semantics by taking one snapshot at the original
// load and copying that snapshot to every destination at the original sites.
func (l *largeAggregateLowerer) transformStoredLoads(m llvm.Module) int {
	l.invalidateLive()
	changed := l.reuseHeaderSnapshots(m)
	var loads []llvm.Value
	var zeroStores []llvm.Value
	var nativeStores []llvm.Value
	slotKind := m.Context().MDKindID(WasmBoundarySlotMD)
	checkNativeResults := l.copyMinSize == 0 && l.roots && l.wasm
	for fn := m.FirstFunction(); !fn.IsNil(); fn = llvm.NextFunction(fn) {
		for bb := fn.FirstBasicBlock(); !bb.IsNil(); bb = llvm.NextBasicBlock(bb) {
			for instr := bb.FirstInstruction(); !instr.IsNil(); instr = llvm.NextInstruction(instr) {
				if store := instr.IsAStoreInst(); !store.IsNil() {
					value := store.Operand(0)
					if value.IsNull() && l.isLargeCopy(value.Type()) {
						zeroStores = append(zeroStores, store)
					} else if checkNativeResults && l.isWasmHeapResultStore(store, slotKind) {
						nativeStores = append(nativeStores, store)
					}
				}
				load := instr.IsALoadInst()
				if !load.IsNil() && l.isLargeCopy(load.Type()) {
					if _, _, ok := l.aggregateUsers(load, false); ok {
						loads = append(loads, load)
					}
				}
			}
		}
	}
	l.prepareWasmResultSlots(m, nativeStores)
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
	return changed + len(loads) + len(zeroStores)
}

// Keep oversized projections of ordinary loads in SSA so LLVM can discard
// unused fields. A whole-value store needs the complete aggregate, and
// private storage already holds a stable copy.
func (l largeAggregateLowerer) aggregateUsers(value llvm.Value, privateStorage bool) (stores, extracts []llvm.Value, ok bool) {
	for use := value.FirstUse(); !use.IsNil(); use = use.NextUse() {
		user := use.User()
		if store := user.IsAStoreInst(); !store.IsNil() && store.Operand(0) == value {
			stores = append(stores, store)
		} else if extract := user.IsAExtractValueInst(); !extract.IsNil() {
			extracts = append(extracts, extract)
		} else {
			return nil, nil, false
		}
	}
	if !privateStorage && len(stores) == 0 {
		for _, extract := range extracts {
			if l.isLargeAggregate(extract.Type()) {
				return nil, nil, false
			}
		}
	}
	return stores, extracts, len(stores)+len(extracts) != 0
}

func (l *largeAggregateLowerer) transformStoredLoad(m llvm.Module, load llvm.Value) {
	stores, extracts, ok := l.aggregateUsers(load, false)
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
		l.replaceTrackedUse(load, copy)
		l.replaceTrackedUse(stores[0], copy)
		stores[0].EraseFromParentAsInstruction()
		load.EraseFromParentAsInstruction()
		return
	}
	snapshot, heap := l.allocSnapshot(m, ctx, b, load, typ, stores, extracts)
	b.SetInsertPointBefore(load)
	if heap {
		// Heap snapshots are a new safepoint absent from the frontend root plan.
		// Keep the source alive before allocating; reflection wrappers can have
		// no original allocation at all.
		l.sourceRoots = append(l.sourceRoots, aggregateRoot{value: load.Operand(0), before: snapshot})
	} else {
		l.callLifetime(ctx, b, true, snapshot)
	}
	copy := l.callMemcpy(ctx, b, snapshot, load.Operand(0), typ)
	setCopyVolatile(ctx, copy, load.IsVolatile())
	copy.InstructionSetDebugLoc(load.InstructionDebugLoc())
	l.replaceTrackedUse(load, copy)
	newUses := l.rewriteMemoryUsers(ctx, load, snapshot, l.td.ABITypeAlignment(typ), stores, extracts)
	if !heap {
		l.markSnapshotLifetimeEnd(ctx, snapshot, copy, newUses)
	}
}

func (l *largeAggregateLowerer) transformCall(m llvm.Module, call llvm.Value) {
	ctx := m.Context()
	oldType := call.CalledFunctionType()
	retType := oldType.ReturnType()
	newType := l.indirectType(ctx, oldType)
	b := ctx.NewBuilder()
	defer b.Dispose()
	b.SetInsertPointBefore(call)

	result := l.allocResult(m, ctx, b, retType, call.InstructionDebugLoc())
	if slot := TakeWasmBoundarySlot(ctx, call); !slot.IsNil() {
		l.useNativeResultStorage(ctx, slot, result)
	}
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

	if !call.FirstUse().IsNil() {
		value := b.CreateLoad(retType, result, "")
		value.InstructionSetDebugLoc(call.InstructionDebugLoc())
		call.ReplaceAllUsesWith(value)
		l.rewriteStoredResult(ctx, value, l.td.ABITypeAlignment(retType))
	}
	call.EraseFromParentAsInstruction()
}

var (
	largeMemoryKind       = int(llvm.AttributeKindID("memory"))
	largeSpeculatableKind = int(llvm.AttributeKindID("speculatable"))
)

func (l *largeAggregateLowerer) transformFunc(m llvm.Module, fn llvm.Value) {
	ctx := m.Context()
	oldType := fn.GlobalValueType()
	retType := oldType.ReturnType()
	newType := l.indirectType(ctx, oldType)
	name := fn.Name()
	fn.SetName("")
	nfn := llvm.AddFunction(m, name, newType)
	nfn.CopyFunctionProperties(fn)
	nfn.AddAttributeAtIndex(1, sretAttribute(ctx, retType))
	for _, attr := range fn.GetFunctionAttributes() {
		// The new sret store is observable, and large snapshots can add
		// allocations and GC-root writes. Let LLVM infer the lowered effects.
		if attr.IsEnum() {
			switch attr.GetEnumKind() {
			case largeMemoryKind, largeSpeculatableKind:
				continue
			}
		}
		nfn.AddFunctionAttr(attr)
	}
	for i := 0; i < oldType.ParamTypesCount(); i++ {
		for _, attr := range fn.GetAttributesAtIndex(i + 1) {
			if !llvmattr.ValidOnSRetParameter(attr) {
				continue
			}
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
	delete(l.debugLocs, fn)
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
			header := l.wasm && isHeaderSnapshot(load)
			if header {
				// Keep the normalized reload visible to snapshot reuse. Store at
				// the original evaluation point, before return-side bookkeeping.
				b.SetInsertPointBefore(llvm.NextInstruction(load))
				b.CreateStore(load, result)
			} else {
				b.SetInsertPointBefore(load)
				l.callMemcpy(ctx, b, result, load.Operand(0), retType)
			}
			b.SetInsertPointBefore(ret)
			b.CreateRetVoid()
			ret.EraseFromParentAsInstruction()
			if !header {
				load.EraseFromParentAsInstruction()
			}
			continue
		}
		b.SetInsertPointBefore(ret)
		b.CreateStore(value, result)
		b.CreateRetVoid()
		ret.EraseFromParentAsInstruction()
	}
}

func (l *largeAggregateLowerer) rewriteStoredResult(ctx llvm.Context, value llvm.Value, alignment int) {
	stores, extracts, ok := l.aggregateUsers(value, true)
	if !ok && !value.FirstUse().IsNil() {
		return
	}
	for _, use := range l.rewriteMemoryUsers(ctx, value, value.Operand(0), alignment, stores, extracts) {
		if !use.IsALoadInst().IsNil() && l.td.TypeAllocSize(use.Type()) >= MinAggregateCopySize {
			l.rewriteStoredResult(ctx, use, use.Alignment())
		}
	}
}

// Root publication extracts pointer members from aggregate SSA values. Rewrite
// those projections as loads from the same immutable snapshot as its stores;
// leaving even one aggregate use expands the entire value in SelectionDAG.
func (l *largeAggregateLowerer) rewriteMemoryUsers(ctx llvm.Context, value, result llvm.Value, baseAlignment int, stores, extracts []llvm.Value) []llvm.Value {
	typ := value.Type()
	b := ctx.NewBuilder()
	defer b.Dispose()
	uses := make([]llvm.Value, 0, len(stores)+len(extracts))
	for _, store := range stores {
		b.SetInsertPointBefore(store)
		copy := l.callMemcpy(ctx, b, store.Operand(1), result, typ)
		setCopyVolatile(ctx, copy, store.IsVolatile())
		copy.InstructionSetDebugLoc(store.InstructionDebugLoc())
		l.replaceTrackedUse(store, copy)
		store.EraseFromParentAsInstruction()
		uses = append(uses, copy)
	}
	for _, extract := range extracts {
		b.SetInsertPointBefore(extract)
		indices := []llvm.Value{llvm.ConstInt(ctx.Int32Type(), 0, false)}
		fieldType := typ
		var offset uint64
		for _, index := range extract.Indices() {
			indices = append(indices, llvm.ConstInt(ctx.Int32Type(), uint64(index), false))
			if fieldType.TypeKind() == llvm.StructTypeKind {
				offset += l.td.ElementOffset(fieldType, int(index))
				fieldType = fieldType.StructElementTypes()[index]
			} else {
				fieldType = fieldType.ElementType()
				offset += uint64(index) * l.td.TypeAllocSize(fieldType)
			}
		}
		field := b.CreateInBoundsGEP(typ, result, indices, "")
		projected := b.CreateLoad(extract.Type(), field, "")
		// A packed ancestor can reduce the alignment of this base address.
		align := baseAlignment
		for offset%uint64(align) != 0 {
			align /= 2
		}
		projected.SetAlignment(align)
		projected.InstructionSetDebugLoc(extract.InstructionDebugLoc())
		extract.ReplaceAllUsesWith(projected)
		l.replaceTrackedUse(extract, projected)
		extract.EraseFromParentAsInstruction()
		uses = append(uses, projected)
	}
	value.EraseFromParentAsInstruction()
	return uses
}

// replaceTrackedUse updates both the snapshot being rewritten and earlier
// snapshots whose live ranges include the replaced instruction. A stale
// range can make a live slot appear free when block and execution order differ.
func (l *largeAggregateLowerer) replaceTrackedUse(old, neu llvm.Value) {
	if old.IsNil() || neu.IsNil() {
		return
	}
	for i := range l.stackSlots {
		slot := &l.stackSlots[i]
		for j := range slot.ranges {
			r := &slot.ranges[j]
			if r.start == old {
				r.start = neu
				r.live = nil
			}
			for k, use := range r.uses {
				if use == old {
					r.uses[k] = neu
					r.live = nil
				}
			}
		}
	}
}

func setCopyVolatile(ctx llvm.Context, copy llvm.Value, volatile bool) {
	if volatile {
		copy.SetOperand(3, llvm.ConstInt(ctx.Int1Type(), 1, false))
	}
}

func (l *largeAggregateLowerer) allocSnapshot(m llvm.Module, ctx llvm.Context, b llvm.Builder, load llvm.Value, typ llvm.Type, stores, extracts []llvm.Value) (llvm.Value, bool) {
	if l.td.TypeAllocSize(typ) >= MinAggregateCopySize {
		return l.allocResult(m, ctx, b, typ, load.InstructionDebugLoc()), true
	}
	fn := load.InstructionParent().Parent()
	l.ensureCFG(fn)
	newRange := snapshotRange{start: load, uses: slices.Concat(stores, extracts)}
	for i := range l.stackSlots {
		slot := &l.stackSlots[i]
		if slot.fn != fn || slot.typ != typ {
			continue
		}
		if l.snapshotRangesInterfere(slot.ranges, &newRange) {
			continue
		}
		slot.ranges = append(slot.ranges, newRange)
		return slot.ptr, false
	}
	ptr := l.allocaAtEntry(b, typ)
	l.stackSlots = append(l.stackSlots, stackSnapshotSlot{
		ptr:    ptr,
		typ:    typ,
		fn:     fn,
		ranges: []snapshotRange{newRange},
	})
	return ptr, false
}

func (l *largeAggregateLowerer) markSnapshotLifetimeEnd(ctx llvm.Context, snapshot, start llvm.Value, uses []llvm.Value) {
	r := snapshotRange{start: start, uses: uses}
	b := ctx.NewBuilder()
	defer b.Dispose()
	for _, use := range uses {
		if !l.isLastSnapshotUse(&r, use) {
			continue
		}
		next := llvm.NextInstruction(use)
		if next.IsNil() {
			continue
		}
		b.SetInsertPointBefore(next)
		l.callLifetime(ctx, b, false, snapshot)
	}
}

func (l largeAggregateLowerer) callLifetime(ctx llvm.Context, b llvm.Builder, start bool, ptr llvm.Value) {
	name := "llvm.lifetime.end"
	if start {
		name = "llvm.lifetime.start"
	}
	// LLVM 22 dropped the i64 size operand; the intrinsic is alloca-only.
	b.CreateIntrinsic(ctx.VoidType(), llvm.LookupIntrinsicID(name), []llvm.Value{ptr}, "")
}

// allocaAtEntry keeps the slot in the entry block so a loop reuses one
// reservation instead of allocating per iteration.
func (l *largeAggregateLowerer) allocaAtEntry(b llvm.Builder, typ llvm.Type) llvm.Value {
	bb := b.GetInsertBlock()
	fn := bb.Parent()
	entry := fn.FirstBasicBlock()
	first := entry.FirstInstruction()
	if first.IsNil() {
		b.SetInsertPointAtEnd(entry)
	} else {
		b.SetInsertPointBefore(first)
	}
	return b.CreateAlloca(typ, "")
}

func (l *largeAggregateLowerer) allocResult(m llvm.Module, ctx llvm.Context, b llvm.Builder, typ llvm.Type, loc llvm.Metadata) llvm.Value {
	intType := ctx.IntType(l.goWordSize * 8)
	ptrType := llvm.PointerType(ctx.Int8Type(), 0)
	size := llvm.ConstInt(intType, l.td.TypeAllocSize(typ), false)
	synthetic := false
	if loc.IsNil() {
		fn := b.GetInsertBlock().Parent()
		if l.debugLocs == nil {
			l.debugLocs = make(map[llvm.Value]debugLocation)
		}
		if cached, ok := l.debugLocs[fn]; ok {
			loc = cached.loc
			synthetic = cached.synthetic
			if cached.synthetic {
				b.SetCurrentDebugLocation(cached.line, 0, cached.scope, llvm.Metadata{})
			}
		} else {
			// The source instruction can be synthetic and have no !dbg even
			// when its function has debug info. Reuse a location from the same
			// function, or create a location at the subprogram line with column
			// zero so LLVM's inlinable-call verifier accepts the call.
			for bb := fn.FirstBasicBlock(); !bb.IsNil() && loc.IsNil(); bb = llvm.NextBasicBlock(bb) {
				for instr := bb.FirstInstruction(); !instr.IsNil(); instr = llvm.NextInstruction(instr) {
					loc = instr.InstructionDebugLoc()
					if !loc.IsNil() {
						break
					}
				}
			}
			line := uint(0)
			var scope llvm.Metadata
			if loc.IsNil() {
				if sp := fn.Subprogram(); !sp.IsNil() {
					line, scope, synthetic = sp.SubprogramLine(), sp, true
					// The builder creates a valid DILocation for the call below.
					b.SetCurrentDebugLocation(line, 0, scope, llvm.Metadata{})
				}
			}
			l.debugLocs[fn] = debugLocation{loc: loc, synthetic: synthetic, line: line, scope: scope}
		}
	}
	align := uint64(l.td.ABITypeAlignment(typ))
	var result llvm.Value
	if align > uint64(l.goWordSize) {
		// The aligned fallback may return an interior pointer; do not attach
		// allocation-base contracts used by heap-to-stack analysis.
		fnType := llvm.FunctionType(ptrType, []llvm.Type{intType, intType}, false)
		fn := m.NamedFunction(runtimeAllocUAligned)
		if fn.IsNil() {
			fn = llvm.AddFunction(m, runtimeAllocUAligned, fnType)
		}
		result = b.CreateCall(fnType, fn, []llvm.Value{size, llvm.ConstInt(intType, align, false)}, "")
	} else {
		fnType := llvm.FunctionType(ptrType, []llvm.Type{intType}, false)
		fn := m.NamedFunction(runtimeAllocU)
		if fn.IsNil() {
			fn = llvm.AddFunction(m, runtimeAllocU, fnType)
		}
		llvmattr.ApplyAllocatorFunction(ctx, fn)
		result = b.CreateCall(fnType, fn, []llvm.Value{size}, "")
		llvmattr.ApplyAllocatorCall(ctx, result, fn, []llvm.Value{size}, l.roots)
	}
	if !synthetic {
		result.InstructionSetDebugLoc(loc)
	} else {
		// Do not let the synthetic location leak to later builder instructions.
		b.SetCurrentDebugLocation(0, 0, llvm.Metadata{}, llvm.Metadata{})
	}
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

func (l *largeAggregateLowerer) invalidateLive() {
	for i := range l.stackSlots {
		for j := range l.stackSlots[i].ranges {
			l.stackSlots[i].ranges[j].live = nil
		}
	}
}

func (l *largeAggregateLowerer) snapshotRangesInterfere(ranges []snapshotRange, next *snapshotRange) bool {
	for i := range ranges {
		if l.liveAt(&ranges[i], next.start) || l.liveAt(next, ranges[i].start) {
			return true
		}
	}
	return false
}

func (l *largeAggregateLowerer) ensureCFG(fn llvm.Value) {
	if l.cfgFn != fn || l.bbPreds == nil {
		l.cfgFn = fn
		l.buildCFG(fn)
	}
}

func (l *largeAggregateLowerer) buildCFG(fn llvm.Value) {
	l.bbs = nil
	l.bbIndex = make(map[llvm.BasicBlock]int)
	for bb := fn.FirstBasicBlock(); !bb.IsNil(); bb = llvm.NextBasicBlock(bb) {
		l.bbIndex[bb] = len(l.bbs)
		l.bbs = append(l.bbs, bb)
	}
	nb := len(l.bbs)
	l.bbPreds = make([][]int, nb)
	for i, bb := range l.bbs {
		term := bb.LastInstruction()
		if term.IsNil() {
			continue
		}
		for s := 0; s < term.SuccessorsCount(); s++ {
			if j, ok := l.bbIndex[term.Successor(s)]; ok {
				l.bbPreds[j] = append(l.bbPreds[j], i)
			}
		}
	}
}

// liveAt reports whether this snapshot's storage is still needed at p.
// Paths that re-enter start are a later iteration of the same slot, so they
// do not keep the current occupant live.
func (l *largeAggregateLowerer) liveAt(r *snapshotRange, p llvm.Value) bool {
	if p.IsNil() {
		return false
	}
	if p == r.start {
		return true
	}
	l.ensureLive(r)
	_, ok := r.live[p]
	return ok
}

func (l *largeAggregateLowerer) ensureLive(r *snapshotRange) {
	if r.live != nil {
		return
	}
	r.live = l.computeLive(r.start, r.uses)
}

func (l *largeAggregateLowerer) computeLive(start llvm.Value, uses []llvm.Value) map[llvm.Value]struct{} {
	live := make(map[llvm.Value]struct{})
	if start.IsNil() {
		return live
	}
	live[start] = struct{}{}
	fwd := l.walkForward(start, start)
	bwd := l.walkUses(uses, start)
	for v := range fwd {
		if _, ok := bwd[v]; ok {
			live[v] = struct{}{}
		}
	}
	return live
}

func (l *largeAggregateLowerer) walkForward(from, avoid llvm.Value) map[llvm.Value]struct{} {
	seen := map[llvm.Value]struct{}{from: {}}
	q := []llvm.Value{from}
	for len(q) > 0 {
		cur := q[0]
		q = q[1:]
		for _, next := range nextInstructions(cur) {
			if !avoid.IsNil() && next == avoid {
				continue
			}
			if _, ok := seen[next]; ok {
				continue
			}
			seen[next] = struct{}{}
			q = append(q, next)
		}
	}
	return seen
}

func (l *largeAggregateLowerer) walkUses(uses []llvm.Value, avoid llvm.Value) map[llvm.Value]struct{} {
	seen := make(map[llvm.Value]struct{})
	q := make([]llvm.Value, 0, len(uses))
	for _, use := range uses {
		if use.IsNil() || (!avoid.IsNil() && use == avoid) {
			continue
		}
		if _, ok := seen[use]; ok {
			continue
		}
		seen[use] = struct{}{}
		q = append(q, use)
	}
	for len(q) > 0 {
		cur := q[0]
		q = q[1:]
		for _, prev := range l.prevInstructions(cur) {
			if !avoid.IsNil() && prev == avoid {
				continue
			}
			if _, ok := seen[prev]; ok {
				continue
			}
			seen[prev] = struct{}{}
			q = append(q, prev)
		}
	}
	return seen
}

func (l *largeAggregateLowerer) prevInstructions(instr llvm.Value) []llvm.Value {
	if prev := llvm.PrevInstruction(instr); !prev.IsNil() {
		return []llvm.Value{prev}
	}
	bb := instr.InstructionParent()
	if bb.IsNil() {
		return nil
	}
	bi, ok := l.bbIndex[bb]
	if !ok || bi >= len(l.bbPreds) {
		return nil
	}
	out := make([]llvm.Value, 0, len(l.bbPreds[bi]))
	for _, pi := range l.bbPreds[bi] {
		term := l.bbs[pi].LastInstruction()
		if !term.IsNil() {
			out = append(out, term)
		}
	}
	return out
}

func (l *largeAggregateLowerer) isLastSnapshotUse(r *snapshotRange, use llvm.Value) bool {
	l.ensureLive(r)
	for _, next := range nextInstructions(use) {
		if next == r.start {
			continue
		}
		if _, ok := r.live[next]; ok {
			return false
		}
	}
	return true
}

func nextInstructions(instr llvm.Value) []llvm.Value {
	if next := llvm.NextInstruction(instr); !next.IsNil() {
		return []llvm.Value{next}
	}
	n := instr.SuccessorsCount()
	if n == 0 {
		return nil
	}
	succs := make([]llvm.Value, 0, n)
	for i := 0; i < n; i++ {
		first := instr.Successor(i).FirstInstruction()
		if !first.IsNil() {
			succs = append(succs, first)
		}
	}
	return succs
}
