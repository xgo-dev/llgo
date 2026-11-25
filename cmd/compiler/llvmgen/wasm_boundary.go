package llvmgen

import (
	llabi "github.com/xgo-dev/llgo/cmd/compiler/abi"
	"github.com/xgo-dev/llvm"
)

// Match ABI copy lowering when choosing between SSA and memory. The ABI can
// use its private byval argument or sret result slot to clear headers in place.
func (b Builder) normalizeWasmBoundaryValue(value llvm.Value) llvm.Value {
	typ := value.Type()
	switch n := b.Prog.wasmHeaderCount(typ, maxInlineWasmHeaders+1); {
	case n == 0:
		return value
	case !b.Prog.needsWasmHeaderStorage(typ, n):
		return b.Prog.clearWasmABIPadding(b.impl, value)
	}
	slot := b.Func.entryAllocaBuilder().CreateAlloca(typ, "")
	store := b.impl.CreateStore(value, slot)
	ctx := b.Prog.ctx
	store.SetMetadata(ctx.MDKindID(llabi.WasmBoundarySlotMD), ctx.MDNode(nil))
	b.clearWasmHeaderRange(typ, slot, llvm.ConstInt(b.Prog.tyInt32(), 1, false))
	load := b.impl.CreateLoad(typ, slot, "")
	// Small boundary snapshots use private stack storage. Large results already
	// have separate lowering for the scanned sret buffer.
	if b.Prog.td.TypeAllocSize(typ) < llabi.MinAggregateCopySize {
		load.SetMetadata(ctx.MDKindID(llabi.HeaderSnapshotMD), ctx.MDNode(nil))
	}
	return load
}

// The data slot and its high word have ordinary LLVM types. Keep their header
// provenance across projections so the native pointer can be used independently.
type wasmPaddingProjection struct {
	typ         llvm.Type
	slotPadding bool
}

func (p Program) wasmPaddingObserved(value llvm.Value) bool {
	return p.wasmPaddingValueUsed(value, wasmPaddingProjection{typ: value.Type()}, make(map[llvm.Value]bool))
}

func (p Program) wasmPaddingValueUsed(value llvm.Value, field wasmPaddingProjection, visiting map[llvm.Value]bool) bool {
	if !field.slotPadding && !p.hasWasmABIPadding(field.typ) {
		return false
	}
	if visiting[value] {
		return true
	}
	visiting[value] = true
	defer delete(visiting, value)
	for use := value.FirstUse(); !use.IsNil(); use = use.NextUse() {
		user := use.User()
		switch {
		case !user.IsAStoreInst().IsNil():
			slot := user.Operand(1).IsAAllocaInst()
			if user.IsVolatile() || user.Operand(0) != value || slot.IsNil() ||
				slot.AllocatedType() != field.typ || slot.Operand(0).IsAConstantInt().IsNil() ||
				slot.Operand(0).ZExtValue() != 1 || p.wasmPaddingRead(slot, field, user, visiting) {
				return true
			}
		case !user.IsAExtractValueInst().IsNil():
			projected := field
			for _, index := range user.Indices() {
				projected = p.projectWasmPadding(projected, uint64(index))
			}
			if p.wasmPaddingValueUsed(user, projected, visiting) {
				return true
			}
		case !user.IsAInsertValueInst().IsNil() && user.Operand(0) == value:
			// Replacing the high word (or its enclosing data slot) kills
			// the incoming padding. The inserted operand has its own uses.
			indices := user.Indices()
			replacesHeaderPadding := p.isWasmABIPaddedHeader(field.typ) && indices[0] == 0 &&
				(len(indices) == 1 || indices[1] == 1)
			replacesSlotPadding := field.slotPadding && indices[0] == 1
			if !replacesHeaderPadding && !replacesSlotPadding {
				return true
			}
		default:
			return true
		}
	}
	return false
}

// Parameter homes can hide projections behind a store, GEP and load. Follow
// only a private, singly initialized slot; an escape, mutation, reinterpreted
// load or unknown offset keeps normalization. No alias analysis is needed.
func (p Program) wasmPaddingRead(ptr llvm.Value, field wasmPaddingProjection, init llvm.Value, visiting map[llvm.Value]bool) bool {
	if visiting[ptr] {
		return true
	}
	visiting[ptr] = true
	defer delete(visiting, ptr)
	for use := ptr.FirstUse(); !use.IsNil(); use = use.NextUse() {
		user := use.User()
		switch {
		case user == init, !user.IsAICmpInst().IsNil():
			// Neither initialization nor comparing the address reads padding.
		case !user.IsALoadInst().IsNil():
			if user.Type() != field.typ || user.IsVolatile() || user.Ordering() != llvm.AtomicOrderingNotAtomic ||
				p.wasmPaddingValueUsed(user, field, visiting) {
				return true
			}
		case !user.IsAGetElementPtrInst().IsNil():
			projected := field
			for i := 1; i < user.OperandsCount(); i++ {
				index := user.Operand(i).IsAConstantInt()
				if index.IsNil() {
					return true
				}
				if i == 1 && user.GEPSourceElementType() == field.typ {
					if !index.IsNull() {
						return true
					}
					continue
				}
				// Array indexing can start directly from the element type.
				if i == 1 && (field.typ.TypeKind() != llvm.ArrayTypeKind || user.GEPSourceElementType() != field.typ.ElementType()) {
					return true
				}
				projected = p.projectWasmPadding(projected, index.ZExtValue())
				if projected.typ.IsNil() {
					return true
				}
			}
			if p.wasmPaddingRead(user, projected, init, visiting) {
				return true
			}
		case !user.IsACallInst().IsNil() && user.Operand(0) == ptr:
			callee := user.CalledValue().IsAFunction()
			if callee.IsNil() {
				return true
			}
			switch callee.IntrinsicID() {
			case llvm.LookupIntrinsicID("llvm.memset"):
				if !user.Operand(3).IsNull() {
					return true
				}
			case llvm.LookupIntrinsicID("llvm.lifetime.start"), llvm.LookupIntrinsicID("llvm.lifetime.end"):
			default:
				return true
			}
		default:
			return true
		}
	}
	return false
}

func (p Program) projectWasmPadding(field wasmPaddingProjection, index uint64) wasmPaddingProjection {
	switch field.typ.TypeKind() {
	case llvm.ArrayTypeKind:
		if index < uint64(field.typ.ArrayLength()) {
			return wasmPaddingProjection{typ: field.typ.ElementType()}
		}
	case llvm.StructTypeKind:
		if fields := field.typ.StructElementTypes(); index < uint64(len(fields)) {
			slotPadding := p.isWasmABIPaddedHeader(field.typ) && index == 0 || field.slotPadding && index == 1
			return wasmPaddingProjection{typ: fields[index], slotPadding: slotPadding}
		}
	}
	return wasmPaddingProjection{}
}

func (f Function) finishWasmABIBoundaries() {
	if !f.Prog.usesWideGoStorage() {
		return
	}
	// Application uses and SSA root projections are complete. The GC prologue
	// added afterwards publishes stack field addresses with native-pointer
	// metadata, so those indirect reads cannot observe a header's high word.
	var b Builder
	normalize := func(value, before llvm.Value) {
		if !f.Prog.wasmPaddingObserved(value) {
			return
		}
		if b == nil {
			b = f.NewBuilder()
		}
		b.impl.SetInsertPointBefore(before)
		placeholder := b.impl.CreatePHI(value.Type(), "")
		value.ReplaceAllUsesWith(placeholder)
		normalized := b.normalizeWasmBoundaryValue(value)
		placeholder.ReplaceAllUsesWith(normalized)
		placeholder.EraseFromParentAsInstruction()
	}
	// A C callback can call any environment-free Go entry. Include parameters
	// obtained before the body was built, as well as native call results.
	if f.env == nil || f.Prog.isNativeStorage(f.Type) {
		for i := range f.params {
			normalize(f.Param(i).impl, f.impl.FirstBasicBlock().FirstInstruction())
		}
	}
	for _, result := range f.wasmABIResults {
		normalize(result, llvm.NextInstruction(result))
	}
	f.wasmABIResults = nil
	if b != nil {
		b.Dispose()
	}
}
