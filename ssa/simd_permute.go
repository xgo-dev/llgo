package ssa

import "github.com/xgo-dev/llvm"

func (b Builder) simdPermute(op SIMDOp, args []Expr) Expr {
	x, indices := args[0], args[len(args)-1]
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
	group := n
	if op == SIMDPermuteOrZeroGrouped {
		group = 16
	}
	mask := llvm.ConstInt(indices.ll.ElementType(), uint64(group-1), false)
	result := llvm.Undef(x.ll)
	for i := 0; i < n; i++ {
		lane := llvm.ConstInt(b.Prog.tyInt32(), uint64(i), false)
		index := b.impl.CreateExtractElement(indices.impl, lane, "")
		safe := b.impl.CreateAnd(index, mask, "")
		if op == SIMDPermuteOrZeroGrouped {
			safe = b.impl.CreateOr(safe, llvm.ConstInt(safe.Type(), uint64(i/group*group), false), "")
		}
		value := b.impl.CreateExtractElement(x.impl, safe, "")
		if op == SIMDConcatPermute {
			other := b.impl.CreateExtractElement(args[1].impl, safe, "")
			sourceBit := b.impl.CreateAnd(index, llvm.ConstInt(index.Type(), uint64(n), false), "")
			fromY := llvm.CreateICmp(b.impl, llvm.IntNE, sourceBit, llvm.ConstNull(index.Type()))
			value = b.impl.CreateSelect(fromY, other, value, "")
		}
		if op == SIMDPermuteOrZero || op == SIMDPermuteOrZeroGrouped {
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
		group = 16 / int(b.Prog.td.TypeAllocSize(x.ll.ElementType()))
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

// Keep pack operations visible to instruction selection. LLVM legalizes these
// operations using the features of each physical function after FMV cloning.
func (b Builder) simdPack(op SIMDOp, x, mask Expr) Expr {
	active := llvm.CreateICmp(b.impl, llvm.IntNE, mask.impl, llvm.ConstNull(mask.ll))
	zero := llvm.ConstNull(x.ll)
	if op == SIMDCompress {
		// Compress moves lane bits. Integer carriers also let LLVM 22
		// scalarize floating-point lanes on targets without packed compress.
		bits := b.impl.CreateBitCast(x.impl, mask.ll, "")
		v := b.impl.CreateIntrinsic(mask.ll, llvm.LookupIntrinsicID("llvm.experimental.vector.compress"), []llvm.Value{bits, active, llvm.ConstNull(mask.ll)}, "")
		return Expr{b.impl.CreateBitCast(v, x.ll, ""), x.Type}
	}
	// Expand consumes consecutive elements of the input value. Use a private
	// copy so a computed vector and a vector loaded from memory behave alike.
	slot := llvm.CreateAlloca(b.Func.entryAllocaBuilder(), x.ll)
	b.impl.CreateStore(x.impl, slot)
	v := b.impl.CreateIntrinsic(x.ll, llvm.LookupIntrinsicID("llvm.masked.expandload"), []llvm.Value{slot, active, zero}, "")
	return Expr{v, x.Type}
}
