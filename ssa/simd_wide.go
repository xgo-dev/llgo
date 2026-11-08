package ssa

import "github.com/xgo-dev/llvm"

func (b Builder) simdHalf(op SIMDOp, result Type, args []Expr) Expr {
	x := args[0]
	n := x.ll.VectorSize()
	mask := make([]llvm.Value, result.ll.VectorSize())
	y := llvm.Undef(x.ll)
	if op == SIMDGetLo || op == SIMDGetHi {
		offset := 0
		if op == SIMDGetHi {
			offset = n / 2
		}
		for i := range mask {
			mask[i] = llvm.ConstInt(b.Prog.tyInt32(), uint64(i+offset), false)
		}
	} else {
		// Extend the half to the input vector's size before the final shuffle.
		// The undefined upper lanes are never selected by that shuffle.
		half := args[1]
		pad := make([]llvm.Value, n)
		for i := range pad {
			pad[i] = llvm.ConstInt(b.Prog.tyInt32(), uint64(i), false)
		}
		y = b.impl.CreateShuffleVector(half.impl, llvm.Undef(half.ll), llvm.ConstVector(pad, false), "")
		for i := range mask {
			index := i
			if op == SIMDSetLo && i < n/2 {
				index = n + i
			}
			if op == SIMDSetHi && i >= n/2 {
				index = n + i - n/2
			}
			mask[i] = llvm.ConstInt(b.Prog.tyInt32(), uint64(index), false)
		}
	}
	return Expr{b.impl.CreateShuffleVector(x.impl, y, llvm.ConstVector(mask, false), ""), result}
}
