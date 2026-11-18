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
	// Dynamic amd64 permutations use a baseline-safe lane gather; constant
	// indices fold to shufflevector during LLVM optimization.
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

// simdInterleave only selects existing lanes: no integer arithmetic, poison
// padding or target-specific intrinsic is needed. Grouped operations stay
// within each corresponding 128-bit pair of input groups.
func (b Builder) simdInterleave(op SIMDOp, x, y Expr) Expr {
	n := x.ll.VectorSize()
	group := n
	if op == SIMDInterleaveLoGrouped || op == SIMDInterleaveHiGrouped {
		// The frontend restricts Grouped declarations to 256 bits.
		group /= 2
	}
	mask := make([]llvm.Value, n)
	for base := 0; base < n; base += group {
		for i := 0; i < group; i++ {
			var index int
			switch op {
			case SIMDInterleaveLo, SIMDInterleaveLoGrouped:
				index = base + i/2 + (i%2)*n
			case SIMDInterleaveHi, SIMDInterleaveHiGrouped:
				index = base + i/2 + (i%2)*n + group/2
			case SIMDInterleaveEven, SIMDInterleaveOdd:
				index = base + 2*(i/2) + (i%2)*n
				if op == SIMDInterleaveOdd {
					index++
				}
			case SIMDConcatEven, SIMDConcatOdd:
				index = base + 2*(i%(group/2)) + (i/(group/2))*n
				if op == SIMDConcatOdd {
					index++
				}
			}
			mask[base+i] = llvm.ConstInt(b.Prog.tyInt32(), uint64(index), false)
		}
	}
	return Expr{b.impl.CreateShuffleVector(x.impl, y.impl, llvm.ConstVector(mask, false), ""), x.Type}
}
