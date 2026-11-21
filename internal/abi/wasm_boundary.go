package abi

import (
	"strings"

	"github.com/xgo-dev/llvm"
)

// WasmBoundarySlotMD identifies the private storage used to normalize an
// incoming by-value parameter or native call result on Memory32.
const WasmBoundarySlotMD = "llgo.wasm.boundary.slot"

// TakeWasmBoundarySlot consumes the frontend's initializing store when the
// ABI already supplies private byval or sret storage for the same value.
func TakeWasmBoundarySlot(ctx llvm.Context, value llvm.Value) llvm.Value {
	use := value.FirstUse()
	if use.IsNil() || !use.NextUse().IsNil() {
		return llvm.Value{}
	}
	store := use.User()
	if store.IsAStoreInst().IsNil() || store.IsVolatile() || store.Operand(0) != value ||
		!store.HasMetadata() || store.Metadata(ctx.MDKindID(WasmBoundarySlotMD)).IsNil() {
		return llvm.Value{}
	}
	if !value.IsACallInst().IsNil() && discardUnusedBoundaryStore(store) {
		return llvm.Value{}
	}
	slot := store.Operand(1)
	store.EraseFromParentAsInstruction()
	return slot
}

// A discarded boundary value needs only the original call. Remove the
// private store, range clear and reload together before allocating sret.
func discardUnusedBoundaryStore(store llvm.Value) bool {
	slot := store.Operand(1)
	clear := llvm.NextInstruction(store)
	if slot.IsAAllocaInst().IsNil() || clear.IsNil() || clear.IsACallInst().IsNil() ||
		!strings.HasPrefix(clear.CalledValue().Name(), "__llgo_clear_header_range") {
		return false
	}
	load := llvm.NextInstruction(clear)
	if load.IsNil() || load.IsALoadInst().IsNil() || load.IsVolatile() || !load.FirstUse().IsNil() || !isPrivateHeaderSnapshot(load) {
		return false
	}
	load.EraseFromParentAsInstruction()
	clear.EraseFromParentAsInstruction()
	store.EraseFromParentAsInstruction()
	slot.EraseFromParentAsInstruction()
	return true
}

func (l *largeAggregateLowerer) isWasmHeapResultStore(store llvm.Value, slotKind int) bool {
	return store.HasMetadata() && !store.Metadata(slotKind).IsNil() &&
		!store.Operand(0).IsACallInst().IsNil() && !store.Operand(1).IsAAllocaInst().IsNil() &&
		l.td.TypeAllocSize(store.Operand(0).Type()) >= MinAggregateCopySize
}

// A later copy of these results can allocate a snapshot. Prepare scanned,
// rooted sret storage before the native call fills in its pointers.
func (l *largeAggregateLowerer) prepareWasmResultSlots(m llvm.Module, stores []llvm.Value) {
	if len(stores) == 0 {
		return
	}
	ctx := m.Context()
	b := ctx.NewBuilder()
	defer b.Dispose()
	for _, store := range stores {
		if discardUnusedBoundaryStore(store) {
			continue
		}
		call := store.Operand(0)
		b.SetInsertPointBefore(call)
		result := l.allocResult(m, ctx, b, call.Type(), call.InstructionDebugLoc())
		l.useNativeResultStorage(ctx, store.Operand(1), result)
	}
}

// After the range clear, this fresh, rooted sret buffer already owns the
// immutable SSA result. Read its projections in place even across safepoints.
func (l *largeAggregateLowerer) useNativeResultStorage(ctx llvm.Context, slot, result llvm.Value) {
	var reload llvm.Value
	for use := slot.FirstUse(); !use.IsNil(); use = use.NextUse() {
		if load := use.User().IsALoadInst(); !load.IsNil() && !load.IsVolatile() {
			reload = load
			break
		}
	}
	slot.ReplaceAllUsesWith(result)
	slot.EraseFromParentAsInstruction()
	if !reload.IsNil() {
		l.rewriteStoredResult(ctx, reload, l.td.ABITypeAlignment(reload.Type()))
	}
}
