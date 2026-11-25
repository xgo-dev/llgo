package llvmgen

import "github.com/xgo-dev/llvm"

func (b Builder) simdStoreMasked(x, ptr, mask Expr) Expr {
	// AMD64 mask stores enable a lane by its sign bit. Inactive lanes must not
	// read or write memory, even when their addresses are inaccessible.
	enabled := llvm.CreateICmp(b.impl, llvm.IntSLT, mask.impl, llvm.ConstNull(mask.ll))
	any := b.impl.CreateIntrinsic(b.Prog.Bool().ll, llvm.LookupIntrinsicID("llvm.vector.reduce.or"), []llvm.Value{enabled}, "")
	logicalBlock, entryBlock := b.blk, b.impl.GetInsertBlock()
	blocks := b.Func.MakeBlocks(2)
	b.If(Expr{any, b.Prog.Bool()}, blocks[0], blocks[1])
	b.SetBlock(blocks[0])
	// A zero mask does not dereference ptr. Active nil accesses still get the
	// same recoverable Go panic as ordinary SIMD loads and stores.
	b.AssertNilDeref(ptr)
	b.Jump(blocks[1])
	b.SetBlockEx(blocks[1], AtEnd, false)
	b.blk = logicalBlock
	if logicalBlock.last == entryBlock {
		logicalBlock.last = blocks[1].last
	}
	// LLVM 22 carries alignment on the pointer argument, not as an operand.
	store := b.impl.CreateIntrinsic(b.Prog.Void().ll, llvm.LookupIntrinsicID("llvm.masked.store"), []llvm.Value{x.impl, ptr.impl, enabled}, "")
	const pointerAttributeIndex = 2 // LLVM argument attributes are one-based.
	store.AddCallSiteAttribute(pointerAttributeIndex, b.Prog.ctx.CreateEnumAttribute(llvm.AttributeKindID("align"), uint64(b.Prog.AlignOf(b.Prog.Elem(ptr.Type)))))
	return Expr{store, b.Prog.Void()}
}

// SIMDLoadPartial reads min(len(slice), lanes) elements and zeroes the rest.
func (b Builder) SIMDLoadPartial(vector Type, slice Expr) (Expr, Expr) {
	length := b.SliceLen(slice)
	lanes := vector.ll.VectorSize()
	limit := llvm.ConstInt(length.ll, uint64(lanes), false)
	count := b.impl.CreateIntrinsic(length.ll, llvm.LookupIntrinsicID("llvm.umin"), []llvm.Value{length.impl, limit}, "")
	maskType := llvm.VectorType(b.Prog.Bool().ll, lanes)
	mask := b.impl.CreateIntrinsic(maskType, llvm.LookupIntrinsicID("llvm.get.active.lane.mask"), []llvm.Value{llvm.ConstNull(length.ll), count}, "")
	ptr := b.SliceData(slice)
	load := b.impl.CreateIntrinsic(vector.ll, llvm.LookupIntrinsicID("llvm.masked.load"), []llvm.Value{ptr.impl, mask, llvm.ConstNull(vector.ll)}, "")
	load.AddCallSiteAttribute(1, b.Prog.ctx.CreateEnumAttribute(llvm.AttributeKindID("align"), uint64(b.Prog.AlignOf(b.Prog.Elem(ptr.Type)))))
	return Expr{load, vector}, Expr{count, length.Type}
}
