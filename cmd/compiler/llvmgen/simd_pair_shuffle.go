package llvmgen

import "github.com/xgo-dev/llvm"

// simdConcatSelected implements the 128/256-bit SHUFPS/SHUFPD lane semantics.
// Each 128-bit output group takes its lower half from x and upper half from y.
// 32-bit lanes reuse the same four two-bit selectors in every group; 64-bit
// lanes consume a distinct selector bit for each output lane, including the
// upper group. These operations move bits, including NaN payloads and -0.
func (b Builder) simdConcatSelected(x, selector, y Expr) Expr {
	n := x.ll.VectorSize()
	elemBits := int(b.Prog.td.TypeAllocSize(x.ll.ElementType()) * 8)
	group := 128 / elemBits
	shift := func(i int) int {
		if elemBits == 32 {
			return (i % group) * 2
		}
		return i
	}
	if c := selector.impl.IsAConstantInt(); !c.IsNil() {
		mask := make([]llvm.Value, n)
		for i := range mask {
			index := i/group*group + (int(c.ZExtValue()>>shift(i)) & (group - 1))
			if i%group >= group/2 {
				index += n
			}
			mask[i] = llvm.ConstInt(b.Prog.tyInt32(), uint64(index), false)
		}
		return Expr{b.impl.CreateShuffleVector(x.impl, y.impl, llvm.ConstVector(mask, false), ""), x.Type}
	}
	result := llvm.Undef(x.ll)
	for i := 0; i < n; i++ {
		index := b.impl.CreateLShr(selector.impl, llvm.ConstInt(selector.ll, uint64(shift(i)), false), "")
		index = b.impl.CreateAnd(index, llvm.ConstInt(selector.ll, uint64(group-1), false), "")
		index = b.impl.CreateZExt(index, b.Prog.tyInt32(), "")
		index = b.impl.CreateAdd(index, llvm.ConstInt(b.Prog.tyInt32(), uint64(i/group*group), false), "")
		input := x.impl
		if i%group >= group/2 {
			input = y.impl
		}
		// Masking precedes extraction, so every dynamic lane index is valid.
		value := b.impl.CreateExtractElement(input, index, "")
		result = b.impl.CreateInsertElement(result, value, llvm.ConstInt(b.Prog.tyInt32(), uint64(i), false), "")
	}
	return Expr{result, x.Type}
}

// simdConcatPermute128 chooses two complete 128-bit halves from {x.lo, x.hi,
// y.lo, y.hi}. Unlike the packed selectors above, lo/hi must be in [0, 4).
func (b Builder) simdConcatPermute128(x, lo, hi, y Expr) Expr {
	indices := []llvm.Value{b.simdCheckedIndex(lo, 4), b.simdCheckedIndex(hi, 4)}
	n := x.ll.VectorSize()
	group := n / 2
	if !indices[0].IsAConstantInt().IsNil() && !indices[1].IsAConstantInt().IsNil() && indices[0].ZExtValue() < 4 && indices[1].ZExtValue() < 4 {
		mask := make([]llvm.Value, n)
		for i := range mask {
			index := int(indices[i/group].ZExtValue())*group + i%group
			mask[i] = llvm.ConstInt(b.Prog.tyInt32(), uint64(index), false)
		}
		return Expr{b.impl.CreateShuffleVector(x.impl, y.impl, llvm.ConstVector(mask, false), ""), x.Type}
	}
	// Select whole 128-bit halves. Dynamic element extraction otherwise makes
	// x86 spill the vectors and reconstruct the result with indexed scalar loads.
	var candidates [4]llvm.Value
	for half := range candidates {
		mask := make([]llvm.Value, group)
		for i := range mask {
			mask[i] = llvm.ConstInt(b.Prog.tyInt32(), uint64(half*group+i), false)
		}
		candidates[half] = b.impl.CreateShuffleVector(x.impl, y.impl, llvm.ConstVector(mask, false), "")
	}
	var halves [2]llvm.Value
	for half, index := range indices {
		fromX := llvm.CreateICmp(b.impl, llvm.IntULT, index, llvm.ConstInt(index.Type(), 2, false))
		lowBit := b.impl.CreateAnd(index, llvm.ConstInt(index.Type(), 1, false), "")
		isHigh := llvm.CreateICmp(b.impl, llvm.IntNE, lowBit, llvm.ConstInt(index.Type(), 0, false))
		xHalf := b.impl.CreateSelect(isHigh, candidates[1], candidates[0], "")
		yHalf := b.impl.CreateSelect(isHigh, candidates[3], candidates[2], "")
		halves[half] = b.impl.CreateSelect(fromX, xHalf, yHalf, "")
	}
	mask := make([]llvm.Value, n)
	for i := range mask {
		mask[i] = llvm.ConstInt(b.Prog.tyInt32(), uint64(i), false)
	}
	return Expr{b.impl.CreateShuffleVector(halves[0], halves[1], llvm.ConstVector(mask, false), ""), x.Type}
}
