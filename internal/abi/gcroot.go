package abi

import (
	"strconv"

	"github.com/xgo-dev/llvm"
)

// GCRootFrame is the shadow-frame layout used by temporaries introduced after
// frontend root planning.
type GCRootFrame struct {
	Frame, Prev llvm.Value
	Slots       []llvm.Value
	storageType llvm.Type
}

// NewGCRootFrame adds an outer frame before the existing entry. When frames
// are nested by successive lowering passes, pop them in the reverse order.
func NewGCRootFrame(m llvm.Module, fn llvm.Value, count, pointerSize, goWordSize int, wasm bool) GCRootFrame {
	ctx := m.Context()
	b := ctx.NewBuilder()
	defer b.Dispose()
	ptr := llvm.PointerType(ctx.Int8Type(), 0)
	storageType := gcRootStorageType(ctx, ptr, pointerSize, goWordSize)
	array := llvm.ArrayType(storageType, count)
	typ := ctx.StructType([]llvm.Type{storageType, storageType, array}, false)
	entry := fn.FirstBasicBlock()
	prologue := ctx.InsertBasicBlock(entry, "gcroot.entry")
	initialize := ctx.InsertBasicBlock(entry, "gcroot.init")
	b.SetInsertPointAtEnd(prologue)
	frame := b.CreateAlloca(typ, "")
	chain := rootGlobal(m, "llvm_gc_root_chain", storageType, goWordSize)
	replay := rootGlobal(m, "llvm_gc_root_sjlj_replaying", ctx.Int1Type(), 1)
	prev := b.CreateLoad(storageType, chain, "")
	prevSlot := b.CreateStructGEP(typ, frame, 0, "")
	reentered := b.CreateICmp(llvm.IntEQ, gcRootPointer(b, prev), frame, "")
	replaying := b.CreateLoad(ctx.Int1Type(), replay, "")
	reuse := b.CreateOr(reentered, replaying, "")
	rootArray := b.CreateStructGEP(typ, frame, 2, "")
	slots := make([]llvm.Value, count)
	for i := range slots {
		slots[i] = b.CreateInBoundsGEP(array, rootArray, []llvm.Value{
			llvm.ConstInt(ctx.Int32Type(), 0, false),
			llvm.ConstInt(ctx.Int32Type(), uint64(i), false),
		}, "")
	}
	b.CreateCondBr(reuse, entry, initialize)
	b.SetInsertPointAtEnd(initialize)
	b.CreateStore(prev, prevSlot)
	b.CreateStore(gcRootStorageValue(b, storageType, rootMap(m, fn, count, wasm)), b.CreateStructGEP(typ, frame, 1, ""))
	for _, slot := range slots {
		b.CreateStore(llvm.ConstNull(storageType), slot)
	}
	b.CreateStore(gcRootStorageValue(b, storageType, frame), chain)
	b.CreateBr(entry)
	return GCRootFrame{Frame: frame, Prev: prevSlot, Slots: slots, storageType: storageType}
}

func gcRootStorageType(ctx llvm.Context, ptr llvm.Type, pointerSize, goWordSize int) llvm.Type {
	if goWordSize <= pointerSize {
		return ptr
	}
	return ctx.StructType([]llvm.Type{
		ptr,
		ctx.IntType((goWordSize - pointerSize) * 8),
	}, false)
}

func gcRootStorageValue(b llvm.Builder, storageType llvm.Type, ptr llvm.Value) llvm.Value {
	if storageType.TypeKind() != llvm.StructTypeKind {
		return ptr
	}
	parts := storageType.StructElementTypes()
	value := b.CreateInsertValue(llvm.Undef(storageType), ptr, 0, "")
	return b.CreateInsertValue(value, llvm.ConstNull(parts[1]), 1, "")
}

func gcRootPointer(b llvm.Builder, stored llvm.Value) llvm.Value {
	if stored.Type().TypeKind() != llvm.StructTypeKind {
		return stored
	}
	return b.CreateExtractValue(stored, 0, "")
}

// StorePointer publishes a physical address in the Go pointer storage format.
func (frame GCRootFrame) StorePointer(b llvm.Builder, slot, ptr llvm.Value) {
	b.CreateStore(gcRootStorageValue(b, frame.storageType, ptr), slot)
}

// PopGCRootFrame restores the chain only when this frame is its actual head.
// Discarded entries replayed by SJLJ may return without having linked a frame.
func PopGCRootFrame(m llvm.Module, fn llvm.Value, frame GCRootFrame) {
	ctx := m.Context()
	b := ctx.NewBuilder()
	defer b.Dispose()
	chain := m.NamedGlobal("llvm_gc_root_chain")
	for bb := fn.FirstBasicBlock(); !bb.IsNil(); bb = llvm.NextBasicBlock(bb) {
		ret := bb.LastInstruction()
		if ret.IsNil() || ret.InstructionOpcode() != llvm.Ret {
			continue
		}
		b.SetInsertPointBefore(ret)
		current := gcRootPointer(b, b.CreateLoad(frame.storageType, chain, ""))
		prev := gcRootPointer(b, b.CreateLoad(frame.storageType, frame.Prev, ""))
		linked := b.CreateICmp(llvm.IntEQ, current, frame.Frame, "")
		restored := b.CreateSelect(linked, prev, current, "")
		b.CreateStore(gcRootStorageValue(b, frame.storageType, restored), chain)
	}
}

func rootGlobal(m llvm.Module, name string, typ llvm.Type, alignment int) llvm.Value {
	g := m.NamedGlobal(name)
	if g.IsNil() {
		g = llvm.AddGlobal(m, typ, name)
	}
	g.SetInitializer(llvm.ConstNull(typ))
	g.SetLinkage(llvm.LinkOnceAnyLinkage)
	g.SetAlignment(alignment)
	return g
}

func rootMap(m llvm.Module, fn llvm.Value, count int, wasm bool) llvm.Value {
	ctx := m.Context()
	name := fn.Name() + "$gcmap"
	if wasm {
		name = "__llgo_wasm_gcmap$" + strconv.Itoa(count)
		if g := m.NamedGlobal(name); !g.IsNil() {
			return g
		}
	}
	typ := ctx.StructType([]llvm.Type{ctx.Int32Type(), ctx.Int32Type()}, false)
	g := llvm.AddGlobal(m, typ, name)
	g.SetInitializer(llvm.ConstNamedStruct(typ, []llvm.Value{
		llvm.ConstInt(ctx.Int32Type(), uint64(count), false),
		llvm.ConstInt(ctx.Int32Type(), 0, false),
	}))
	g.SetGlobalConstant(true)
	g.SetLinkage(llvm.InternalLinkage)
	if wasm {
		g.SetLinkage(llvm.LinkOnceODRLinkage)
		comdat := m.Comdat(name)
		comdat.SetSelectionKind(llvm.AnyComdatSelectionKind)
		g.SetComdat(comdat)
		g.SetUnnamedAddr(true)
	}
	g.SetAlignment(4)
	return g
}
