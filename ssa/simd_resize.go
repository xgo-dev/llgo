package ssa

import (
	"go/types"

	"github.com/xgo-dev/llvm"
)

// simdIntegerResize keeps lane selection and signedness explicit. Generic LLVM
// operations legalize for the baseline without requiring the source ISA's
// optional instructions (in particular AVX512 narrowing on amd64).
func (b Builder) simdIntegerResize(op SIMDOp, result Type, args []Expr) Expr {
	x := args[0]
	unsigned := simdLanes(x.RawType()).Elem().Underlying().(*types.Basic).Info()&types.IsUnsigned != 0
	if op == SIMDExtendLo || op == SIMDExtendHi {
		mask := make([]llvm.Value, result.ll.VectorSize())
		offset := 0
		if op == SIMDExtendHi {
			offset = len(mask)
		}
		for i := range mask {
			mask[i] = llvm.ConstInt(b.Prog.tyInt32(), uint64(i+offset), false)
		}
		v := b.impl.CreateShuffleVector(x.impl, llvm.Undef(x.ll), llvm.ConstVector(mask, false), "")
		if unsigned {
			v = b.impl.CreateZExt(v, result.ll, "")
		} else {
			v = b.impl.CreateSExt(v, result.ll, "")
		}
		return Expr{v, result}
	}

	toUnsigned := simdLanes(result.RawType()).Elem().Underlying().(*types.Basic).Info()&types.IsUnsigned != 0
	bits := result.ll.ElementType().IntTypeWidth()
	upper := uint64(1)<<bits - 1
	lower := uint64(0)
	if !toUnsigned {
		upper >>= 1
		// ConstInt truncates this two's-complement bound to the source lane width.
		lower = ^upper
	}
	narrow := func(v llvm.Value) llvm.Value {
		greater := llvm.IntSGT
		if unsigned {
			greater = llvm.IntUGT
		}
		if !unsigned {
			lo := simdIntegerConstant(x.ll, lower)
			v = b.impl.CreateSelect(b.impl.CreateICmp(llvm.IntSLT, v, lo, ""), lo, v, "")
		}
		hi := simdIntegerConstant(x.ll, upper)
		v = b.impl.CreateSelect(b.impl.CreateICmp(greater, v, hi, ""), hi, v, "")
		return b.impl.CreateTrunc(v, llvm.VectorType(result.ll.ElementType(), x.ll.VectorSize()), "")
	}
	lo := narrow(x.impl)
	hi := llvm.ConstNull(lo.Type())
	if op == SIMDSaturateConcat {
		hi = narrow(args[1].impl)
	}
	mask := make([]llvm.Value, result.ll.VectorSize())
	for i := range mask {
		index := i
		if op != SIMDSaturateConcat && i >= x.ll.VectorSize() {
			index = x.ll.VectorSize()
		}
		mask[i] = llvm.ConstInt(b.Prog.tyInt32(), uint64(index), false)
	}
	return Expr{b.impl.CreateShuffleVector(lo, hi, llvm.ConstVector(mask, false), ""), result}
}
