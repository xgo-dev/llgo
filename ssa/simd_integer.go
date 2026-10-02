package ssa

import (
	"go/types"

	"github.com/xgo-dev/llvm"
)

func simdIntegerConstant(typ llvm.Type, value uint64) llvm.Value {
	if typ.TypeKind() != llvm.VectorTypeKind {
		return llvm.ConstInt(typ, value, false)
	}
	values := make([]llvm.Value, typ.VectorSize())
	for i := range values {
		values[i] = llvm.ConstInt(typ.ElementType(), value, false)
	}
	return llvm.ConstVector(values, false)
}

func (b Builder) simdShift(op SIMDOp, x, count Expr) Expr {
	distance := count.impl
	if op == SIMDShift {
		// ARM64 variable shifts use only the signed low byte of each count.
		bytes := llvm.VectorType(b.Prog.ctx.Int8Type(), x.ll.VectorSize())
		distance = b.impl.CreateSExt(b.impl.CreateTrunc(distance, bytes, ""), x.ll, "")
		negative := llvm.CreateICmp(b.impl, llvm.IntSLT, distance, llvm.ConstNull(x.ll))
		magnitude := b.impl.CreateSelect(negative, llvm.CreateNeg(b.impl, distance), distance, "")
		left := b.simdShiftValue(x, magnitude, false)
		right := b.simdShiftValue(x, magnitude, true)
		return Expr{b.impl.CreateSelect(negative, right, left, ""), x.Type}
	}
	right := op == SIMDShiftRight || op == SIMDShiftAllRight
	return Expr{b.simdShiftValue(x, distance, right), x.Type}
}

func (b Builder) simdShiftValue(x Expr, count llvm.Value, right bool) llvm.Value {
	width := uint64(x.ll.ElementType().IntTypeWidth())
	large := llvm.CreateICmp(b.impl, llvm.IntUGE, count, simdIntegerConstant(count.Type(), width))
	// Clamp before truncation and shifting so no lane can produce poison.
	safe := b.impl.CreateSelect(large, simdIntegerConstant(count.Type(), width-1), count, "")
	if count.Type().TypeKind() != llvm.VectorTypeKind {
		safe = b.impl.CreateTrunc(safe, x.ll.ElementType(), "")
		v := b.impl.CreateInsertElement(llvm.Undef(x.ll), safe, llvm.ConstInt(b.Prog.tyInt32(), 0, false), "")
		safe = b.simdSplatLane0(Expr{v, x.Type}).impl
	}
	unsigned := simdLanes(x.RawType()).Elem().Underlying().(*types.Basic).Info()&types.IsUnsigned != 0
	if right && !unsigned {
		return b.impl.CreateAShr(x.impl, safe, "")
	}
	var shifted llvm.Value
	if right {
		shifted = b.impl.CreateLShr(x.impl, safe, "")
	} else {
		shifted = b.impl.CreateShl(x.impl, safe, "")
	}
	return b.impl.CreateSelect(large, llvm.ConstNull(x.ll), shifted, "")
}

func (b Builder) simdIntegerIntrinsic(op SIMDOp, x, y Expr) Expr {
	name := map[SIMDOp]string{SIMDAddSaturated: "add.sat", SIMDSubSaturated: "sub.sat", SIMDMin: "min", SIMDMax: "max"}[op]
	prefix := "llvm.s"
	if simdLanes(x.RawType()).Elem().Underlying().(*types.Basic).Info()&types.IsUnsigned != 0 {
		prefix = "llvm.u"
	}
	value := b.impl.CreateIntrinsic(x.ll, llvm.LookupIntrinsicID(prefix+name), []llvm.Value{x.impl, y.impl}, "")
	return Expr{value, x.Type}
}
