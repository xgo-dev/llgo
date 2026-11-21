package ssa

import "github.com/xgo-dev/llvm"

// Each pair of 16-byte groups is a 32-byte stream, low bytes from y.
func (b Builder) simdConcatShiftBytes(x, y, shift Expr) Expr {
	count := shift.impl
	if b.Prog.Target().GOARCH == "arm64" {
		count = b.simdCheckedIndex(shift, 16)
	} else {
		count = b.impl.CreateTrunc(count, b.Prog.ctx.Int8Type(), "")
	}
	n := x.ll.VectorSize()
	window := func(lo, hi llvm.Value, offset int) llvm.Value {
		mask := make([]llvm.Value, n)
		for i := range mask {
			j := i%16 + offset
			source := i/16*16 + j%16
			if j >= 16 {
				source += n
			}
			mask[i] = llvm.ConstInt(b.Prog.tyInt32(), uint64(source), false)
		}
		return b.impl.CreateShuffleVector(lo, hi, llvm.ConstVector(mask, false), "")
	}
	lo, hi, zero := y.impl, x.impl, llvm.ConstNull(x.ll)
	// Apply each set bit of the distance to both halves. Constant counts
	// fold through the selects and dynamic counts never index vector storage.
	for step := 1; step <= 16; step *= 2 {
		bit := b.impl.CreateAnd(count, llvm.ConstInt(count.Type(), uint64(step), false), "")
		set := llvm.CreateICmp(b.impl, llvm.IntNE, bit, llvm.ConstNull(count.Type()))
		lo = b.impl.CreateSelect(set, window(lo, hi, step), lo, "")
		if step < 16 {
			hi = b.impl.CreateSelect(set, window(hi, zero, step), hi, "")
		}
	}
	valid := llvm.CreateICmp(b.impl, llvm.IntULT, count, llvm.ConstInt(count.Type(), 32, false))
	return Expr{b.impl.CreateSelect(valid, lo, zero, ""), x.Type}
}
