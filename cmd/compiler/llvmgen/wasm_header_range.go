package llvmgen

import (
	llabi "github.com/xgo-dev/llgo/cmd/compiler/abi"
	"github.com/xgo-dev/llvm"
)

const maxInlineWasmHeaders = 8

// Values handled by aggregate-copy lowering must stay in memory even when
// they contain few headers. Repacking them in SSA would scalarize the copy.
func (p Program) needsWasmHeaderStorage(typ llvm.Type, count int) bool {
	if count == 0 || p.isWasmABIPaddedHeader(typ) {
		return false
	}
	return count > maxInlineWasmHeaders || p.td.TypeAllocSize(typ) >= llabi.MinAggregateCopySize ||
		p.containsWasmHeaderArrayCopy(typ)
}

// Rebuilding an outer value also rebuilds any array whose headers it contains.
// Preserve those array copies through struct fields and single-element wrappers.
func (p Program) containsWasmHeaderArrayCopy(typ llvm.Type) bool {
	switch typ.TypeKind() {
	case llvm.ArrayTypeKind:
		length := typ.ArrayLength()
		if length == 1 {
			return p.containsWasmHeaderArrayCopy(typ.ElementType())
		}
		return llabi.ShouldLowerArrayCopy(length, p.td.TypeAllocSize(typ), p.td.PointerSize()) &&
			p.hasWasmABIPadding(typ.ElementType())
	case llvm.StructTypeKind:
		for _, field := range typ.StructElementTypes() {
			if p.containsWasmHeaderArrayCopy(field) {
				return true
			}
		}
	}
	return false
}

// clearWasmHeaderRange clears only the padding in a copied range. A source
// slice can alias C memory even after it has passed through ordinary Go code.
func (b Builder) clearWasmHeaderRange(elem llvm.Type, data, count llvm.Value) {
	if count.IsNull() {
		return
	}
	size := b.Prog.tyInt32()
	if count.Type() != size {
		count = llvm.CreateTrunc(b.impl, count, size)
	}
	fn := b.Pkg.wasmHeaderRangeFunc(elem)
	llvm.CreateCall(b.impl, fn.GlobalValueType(), fn, []llvm.Value{data, count})
}

// A counted loop keeps the generated code independent of the number of
// elements, including arrays nested in a struct or another array.
func (p Package) wasmHeaderRangeFunc(elem llvm.Type) llvm.Value {
	if fn := p.wasmHeaderRanges[elem]; !fn.IsNil() {
		return fn
	}
	if p.wasmHeaderRanges == nil {
		p.wasmHeaderRanges = make(map[llvm.Type]llvm.Value)
	}
	prog := p.Prog
	word := prog.tyInt32()
	sig := llvm.FunctionType(prog.tyVoid(), []llvm.Type{prog.tyVoidPtr(), word}, false)
	fn := llvm.AddFunction(p.mod, "__llgo_clear_header_range", sig)
	fn.SetLinkage(llvm.InternalLinkage)
	fn.SetUnnamedAddr(true)
	p.wasmHeaderRanges[elem] = fn
	b := prog.ctx.NewBuilder()
	defer b.Dispose()
	entry := prog.ctx.AddBasicBlock(fn, "entry")
	loop := prog.ctx.AddBasicBlock(fn, "loop")
	done := prog.ctx.AddBasicBlock(fn, "done")
	zero := llvm.ConstNull(word)
	b.SetInsertPointAtEnd(entry)
	b.CreateCondBr(llvm.CreateICmp(b, llvm.IntEQ, fn.Param(1), zero), done, loop)
	b.SetInsertPointAtEnd(loop)
	index := b.CreatePHI(word, "i")
	addr := llvm.CreateInBoundsGEP(b, elem, fn.Param(0), []llvm.Value{index})
	p.clearWasmHeaderAt(b, elem, addr)
	next := b.CreateAdd(index, llvm.ConstInt(word, 1, false), "")
	index.AddIncoming([]llvm.Value{zero, next}, []llvm.BasicBlock{entry, loop})
	b.CreateCondBr(llvm.CreateICmp(b, llvm.IntEQ, next, fn.Param(1)), done, loop)
	b.SetInsertPointAtEnd(done)
	b.CreateRetVoid()
	return fn
}

func (p Package) clearWasmHeaderAt(b llvm.Builder, typ llvm.Type, addr llvm.Value) {
	prog := p.Prog
	if prog.isWasmABIPaddedHeader(typ) {
		slot := typ.StructElementTypes()[0]
		data := llvm.CreateStructGEP(b, typ, addr, 0)
		high := llvm.CreateStructGEP(b, slot, data, 1)
		// A containing C record can be packed.
		b.CreateStore(llvm.ConstNull(slot.StructElementTypes()[1]), high).SetAlignment(1)
		return
	}
	if typ.TypeKind() == llvm.ArrayTypeKind {
		fn := p.wasmHeaderRangeFunc(typ.ElementType())
		n := llvm.ConstInt(prog.tyInt32(), uint64(typ.ArrayLength()), false)
		llvm.CreateCall(b, fn.GlobalValueType(), fn, []llvm.Value{addr, n})
		return
	}
	for i, field := range typ.StructElementTypes() {
		if prog.hasWasmABIPadding(field) {
			p.clearWasmHeaderAt(b, field, llvm.CreateStructGEP(b, typ, addr, i))
		}
	}
}
