package ssa

import "github.com/xgo-dev/llvm"

func (b Builder) simdPermute(op SIMDOp, x, indices Expr) Expr {
	if op == SIMDLookupOrZero {
		name := "llvm.aarch64.neon.tbl1"
		if b.Prog.Target().GOARCH == "wasm" {
			name = "llvm.wasm.swizzle"
		}
		value := b.impl.CreateIntrinsic(x.ll, llvm.LookupIntrinsicID(name), []llvm.Value{x.impl, indices.impl}, "")
		return Expr{value, x.Type}
	}
	// Mask indices before extracting a lane, including indices whose sign bit
	// requests zero. An out-of-bounds extract would create LLVM poison.
	n := x.ll.VectorSize()
	mask := llvm.ConstInt(indices.ll.ElementType(), uint64(n-1), false)
	result := llvm.Undef(x.ll)
	for i := 0; i < n; i++ {
		lane := llvm.ConstInt(b.Prog.tyInt32(), uint64(i), false)
		index := b.impl.CreateExtractElement(indices.impl, lane, "")
		safe := b.impl.CreateAnd(index, mask, "")
		value := b.impl.CreateExtractElement(x.impl, safe, "")
		if op == SIMDPermuteOrZero {
			negative := llvm.CreateICmp(b.impl, llvm.IntSLT, index, llvm.ConstNull(index.Type()))
			value = b.impl.CreateSelect(negative, llvm.ConstNull(value.Type()), value, "")
		}
		result = b.impl.CreateInsertElement(result, value, lane, "")
	}
	return Expr{result, x.Type}
}
