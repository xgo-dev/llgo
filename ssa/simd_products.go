package ssa

import (
	"go/types"

	"github.com/xgo-dev/llvm"
)

func (b Builder) simdAlternating(op SIMDOp, result Type, args []Expr) Expr {
	addend := args[len(args)-1]
	target := b.Prog.Target()
	level := target.goArchitectureSetting(target.GOAMD64)
	// LLVM recognizes a shuffle of alternating subtraction and addition as
	// ADDSUB or FMADDSUB. Keep the single-FMA fallback on CPUs where a
	// pair of vector FMAs could become two sets of scalar libcalls.
	if level == "v4" || level == "v3" || (level == "v2" && op == SIMDAddOddSubEven) {
		var sub, add llvm.Value
		if op == SIMDAddOddSubEven {
			sub = b.impl.CreateFSub(args[0].impl, addend.impl, "")
			add = b.impl.CreateFAdd(args[0].impl, addend.impl, "")
		} else {
			fma := func(z llvm.Value) llvm.Value {
				return b.impl.CreateIntrinsic(result.ll, llvm.LookupIntrinsicID("llvm.fma"), []llvm.Value{args[0].impl, args[1].impl, z}, "")
			}
			sub, add = fma(b.impl.CreateFNeg(addend.impl, "")), fma(addend.impl)
		}
		if op == SIMDMulAddOddSubEven {
			sub, add = add, sub
		}
		n := result.ll.VectorSize()
		mask := make([]llvm.Value, n)
		for i := range mask {
			mask[i] = llvm.ConstInt(b.Prog.tyInt32(), uint64(i+(i%2)*n), false)
		}
		return Expr{b.impl.CreateShuffleVector(sub, add, llvm.ConstVector(mask, false), ""), result}
	}
	negate := make([]llvm.Value, result.ll.VectorSize())
	for i := range negate {
		bit := uint64(i & 1)
		// Go 1.27 emits VFMADDSUB for MulAddEvenSubOdd and ADDSUB for
		// AddOddSubEven. Both subtract in even lanes. VFMSUBADD subtracts
		// in odd lanes, reversing the parity in the two FMA doc comments.
		if op != SIMDMulAddOddSubEven {
			bit = 1 - bit
		}
		negate[i] = llvm.ConstInt(b.Prog.ctx.Int1Type(), bit, false)
	}
	signed := Expr{b.impl.CreateSelect(llvm.ConstVector(negate, false), b.impl.CreateFNeg(addend.impl, ""), addend.impl, ""), result}
	if op == SIMDAddOddSubEven {
		return Expr{b.impl.CreateFAdd(args[0].impl, signed.impl, ""), result}
	}
	return b.simdFMA(result, args[0], args[1], signed)
}

func (b Builder) simdIntegerProduct(op SIMDOp, result Type, x, y Expr) Expr {
	n := x.ll.VectorSize()
	if op == SIMDSumOf8AbsDiff || op == SIMDDotProductPairsSaturated {
		// These byte operations have one instruction per supported width.
		// Use only features guaranteed by the compilation target.
		target := b.Prog.Target()
		level := target.goArchitectureSetting(target.GOAMD64)
		var sad, dot string
		switch {
		case n == 16:
			sad = "llvm.x86.sse2.psad.bw"
			if level == "v2" || level == "v3" || level == "v4" {
				dot = "llvm.x86.ssse3.pmadd.ub.sw.128"
			}
		case n == 32 && (level == "v3" || level == "v4"):
			sad, dot = "llvm.x86.avx2.psad.bw", "llvm.x86.avx2.pmadd.ub.sw"
		case n == 64 && level == "v4":
			sad, dot = "llvm.x86.avx512.psad.bw.512", "llvm.x86.avx512.pmaddubs.w.512"
		}
		name := sad
		if op == SIMDDotProductPairsSaturated {
			name = dot
		}
		if name != "" {
			return Expr{b.impl.CreateIntrinsic(result.ll, llvm.LookupIntrinsicID(name), []llvm.Value{x.impl, y.impl}, ""), result}
		}
	}
	switch op {
	case SIMDMulSign:
		zero := llvm.ConstNull(x.ll)
		neg := llvm.CreateICmp(b.impl, llvm.IntSLT, y.impl, zero)
		pos := llvm.CreateICmp(b.impl, llvm.IntSGT, y.impl, zero)
		return Expr{b.impl.CreateSelect(neg, b.impl.CreateNeg(x.impl, ""), b.impl.CreateSelect(pos, x.impl, zero, ""), ""), result}
	case SIMDSumOf8AbsDiff:
		return b.simdSumOf8AbsDiff(result, x, y)
	}
	bits := x.ll.ElementType().IntTypeWidth()
	wide := llvm.VectorType(b.Prog.ctx.IntType(2*bits), n)
	extend := func(v Expr) llvm.Value {
		if simdLanes(v.RawType()).Elem().Underlying().(*types.Basic).Info()&types.IsUnsigned != 0 {
			return b.impl.CreateZExt(v.impl, wide, "")
		}
		return b.impl.CreateSExt(v.impl, wide, "")
	}
	prod := b.impl.CreateMul(extend(x), extend(y), "")
	if op == SIMDMulHigh {
		return Expr{b.impl.CreateTrunc(b.impl.CreateLShr(prod, simdIntegerConstant(wide, uint64(bits)), ""), result.ll, ""), result}
	}
	halves := func(parity int) llvm.Value {
		mask := make([]llvm.Value, n/2)
		for i := range mask {
			mask[i] = llvm.ConstInt(b.Prog.tyInt32(), uint64(2*i+parity), false)
		}
		return b.impl.CreateShuffleVector(prod, llvm.Undef(wide), llvm.ConstVector(mask, false), "")
	}
	even, odd := Expr{halves(0), result}, Expr{halves(1), result}
	if op == SIMDDotProductPairsSaturated {
		return b.simdIntegerIntrinsic(SIMDAddSaturated, even, odd)
	}
	return Expr{b.impl.CreateAdd(even.impl, odd.impl, ""), result}
}

func (b Builder) simdSumOf8AbsDiff(result Type, x, y Expr) Expr {
	gt := llvm.CreateICmp(b.impl, llvm.IntUGT, x.impl, y.impl)
	diff := b.impl.CreateSelect(gt, b.impl.CreateSub(x.impl, y.impl, ""), b.impl.CreateSub(y.impl, x.impl, ""), "")
	sum := llvm.ConstNull(result.ll)
	for lane := 0; lane < 8; lane++ {
		mask := make([]llvm.Value, result.ll.VectorSize())
		for i := range mask {
			mask[i] = llvm.ConstInt(b.Prog.tyInt32(), uint64(8*i+lane), false)
		}
		group := b.impl.CreateShuffleVector(diff, llvm.Undef(x.ll), llvm.ConstVector(mask, false), "")
		sum = b.impl.CreateAdd(sum, b.impl.CreateZExt(group, result.ll, ""), "")
	}
	return Expr{sum, result}
}
