package ssa

import (
	"go/types"
	"strconv"

	"github.com/xgo-dev/llvm"
)

type gcStackObject struct {
	address llvm.Value
	typ     Type
}

// StackRootBytes reserves field-address slots plus a possible frame header.
// Charging the header per object is conservative when objects share a frame.
func (p Program) StackRootBytes(typ Type) uint64 {
	if !p.GCRootsEnabled() {
		return 0
	}
	count := p.GCRootCount(typ)
	if count == 0 {
		return 0
	}
	return uint64(count+2) * uint64(p.GoWordSize())
}

func (b Builder) registerStackRoots(address llvm.Value, typ Type) {
	if b.Prog.GCRootsEnabled() && b.Prog.GCRootCount(typ) != 0 {
		b.Func.gcStackObjects = append(b.Func.gcStackObjects, gcStackObject{address, typ})
	}
}

func (p Function) finalizeGCRoots() {
	count := len(p.gcRootSlots)
	for _, object := range p.gcStackObjects {
		count += p.Prog.GCRootCount(object.typ)
	}
	if count == 0 {
		return
	}
	roots := p.buildGCRoots(count)
	for i, slot := range p.gcRootSlots {
		slot.impl.ReplaceAllUsesWith(roots[i].impl)
		slot.impl.EraseFromParentAsInstruction()
	}
}

func (p Function) stackRootMetadata() llvm.Value {
	width := p.Prog.PointerSize()
	name := "__llgo_gcroot_stack_pointer_" + strconv.Itoa(width)
	global := p.Pkg.mod.NamedGlobal(name)
	if global.IsNil() {
		global = llvm.AddGlobal(p.Pkg.mod, p.Prog.ctx.Int8Type(), name)
		global.SetInitializer(llvm.ConstInt(p.Prog.ctx.Int8Type(), uint64(width), false))
		global.SetGlobalConstant(true)
		global.SetUnnamedAddr(true)
		global.SetLinkage(llvm.InternalLinkage)
		global.SetAlignment(1)
	}
	return global
}

func (p Function) initializeStackRoots(b Builder, roots []Expr) {
	if len(p.gcStackObjects) == 0 {
		return
	}
	move := p.Prog.ctx.NewBuilder()
	defer move.Dispose()
	move.SetInsertPointBefore(p.impl.EntryBasicBlock().FirstInstruction())
	index := 0
	for _, object := range p.gcStackObjects {
		// Every registered object has one fixed alloca per invocation, even
		// when its declaration is conditional or inside a loop. Its address
		// must dominate the frame publication preceding all safepoints.
		if object.address.InstructionParent() != p.impl.EntryBasicBlock() {
			object.address.RemoveFromParentAsInstruction()
			move.Insert(object.address)
		}
		b.zeroinit(Expr{object.address, p.Prog.VoidPtr()}, p.Prog.IntVal(p.Prog.SizeOf(object.typ), p.Prog.Uintptr()))
		p.stackRootOffsets(object.typ, 0, func(offset uint64) {
			address := object.address
			if offset != 0 {
				address = llvm.CreateInBoundsGEP(b.impl, p.Prog.ctx.Int8Type(), address, []llvm.Value{llvm.ConstInt(p.Prog.ctx.Int64Type(), offset, false)})
			}
			b.SetGCRoot(roots[index], Expr{address, p.Prog.VoidPtr()})
			index++
		})
	}
}

// Offsets use backend storage layouts, including native C fields, logical
// 64-bit Go words in Wasm32, interface data and closure environment words.
func (p Function) stackRootOffsets(typ Type, offset uint64, emit func(uint64)) {
	switch typ.kind {
	case vkPtr, vkMap, vkChan, vkString, vkSlice:
		emit(offset)
	case vkEface, vkIface, vkClosure:
		emit(offset + p.Prog.td.ElementOffset(p.Prog.storageType(typ), 1))
	case vkStruct, vkTuple:
		count := 0
		switch raw := typ.raw.Type.Underlying().(type) {
		case *types.Struct:
			count = raw.NumFields()
		case *types.Tuple:
			count = raw.Len()
		}
		for i := 0; i < count; i++ {
			p.stackRootOffsets(p.Prog.Field(typ, i), offset+p.Prog.OffsetOf(typ, i), emit)
		}
	case vkArray:
		raw := typ.raw.Type.Underlying().(*types.Array)
		elem := p.Prog.Index(typ)
		if p.Prog.GCRootCount(elem) == 0 {
			return
		}
		for i := int64(0); i < raw.Len(); i++ {
			p.stackRootOffsets(elem, offset+uint64(i)*p.Prog.SizeOf(elem), emit)
		}
	}
}
