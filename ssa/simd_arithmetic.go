package ssa

import (
	"go/types"

	"github.com/xgo-dev/llvm"
)

func (b Builder) simdAverage(x, y Expr) Expr {
	// ceil((x+y)/2), without overflowing the lane before division. Arithmetic
	// shifting implements the signed variant's rounding toward positive infinity.
	different := b.impl.CreateXor(x.impl, y.impl, "")
	one := simdIntegerConstant(x.ll, 1)
	var half llvm.Value
	if simdLanes(x.RawType()).Elem().Underlying().(*types.Basic).Info()&types.IsUnsigned != 0 {
		half = b.impl.CreateLShr(different, one, "")
	} else {
		half = b.impl.CreateAShr(different, one, "")
	}
	shared := b.impl.CreateAnd(x.impl, y.impl, "")
	round := b.impl.CreateAnd(different, one, "")
	return Expr{b.impl.CreateAdd(b.impl.CreateAdd(shared, half, ""), round, ""), x.Type}
}

func (b Builder) simdMulWidenLo(result Type, x, y Expr) Expr {
	indices := make([]llvm.Value, result.ll.VectorSize())
	for i := range indices {
		indices[i] = llvm.ConstInt(b.Prog.tyInt32(), uint64(i), false)
	}
	mask := llvm.ConstVector(indices, false)
	unsigned := simdLanes(x.RawType()).Elem().Underlying().(*types.Basic).Info()&types.IsUnsigned != 0
	widen := func(v llvm.Value) llvm.Value {
		lo := b.impl.CreateShuffleVector(v, llvm.Undef(v.Type()), mask, "")
		if unsigned {
			return b.impl.CreateZExt(lo, result.ll, "")
		}
		return b.impl.CreateSExt(lo, result.ll, "")
	}
	return Expr{b.impl.CreateMul(widen(x.impl), widen(y.impl), ""), result}
}

func (b Builder) simdCarrylessMultiply(x, y Expr, selector llvm.Value) Expr {
	// AMD64 chooses x's lane with bit 0 and y's lane with bit 4. ARM64's
	// low-half primitive uses the same operation with a constant zero selector.
	one := llvm.ConstInt(selector.Type(), 1, false)
	xIndex := b.impl.CreateAnd(selector, one, "")
	yIndex := b.impl.CreateAnd(b.impl.CreateLShr(selector, llvm.ConstInt(selector.Type(), 4, false), ""), one, "")
	i128 := b.Prog.ctx.IntType(128)
	result := llvm.Undef(x.ll)
	for base := 0; base < x.ll.VectorSize(); base += 2 {
		// Each 128-bit group selects its own pair. In particular, the high
		// group of a SIMD256 vector must not reuse the low group's operands.
		offset := llvm.ConstInt(selector.Type(), uint64(base), false)
		xLane := b.impl.CreateAdd(xIndex, offset, "")
		yLane := b.impl.CreateAdd(yIndex, offset, "")
		a := b.impl.CreateZExt(b.impl.CreateExtractElement(x.impl, xLane, ""), i128, "")
		c := b.impl.CreateZExt(b.impl.CreateExtractElement(y.impl, yLane, ""), i128, "")
		// Widen before multiplying to retain the complete 64x64 polynomial
		// product. LLVM legalizes clmul without requiring optional features.
		product := b.impl.CreateIntrinsic(i128, llvm.LookupIntrinsicID("llvm.clmul"), []llvm.Value{a, c}, "")
		if x.ll.VectorSize() == 2 {
			return Expr{b.impl.CreateBitCast(product, x.ll, ""), x.Type}
		}
		pair := b.impl.CreateBitCast(product, llvm.VectorType(x.ll.ElementType(), 2), "")
		for i := 0; i < 2; i++ {
			lane := llvm.ConstInt(b.Prog.tyInt32(), uint64(i), false)
			value := b.impl.CreateExtractElement(pair, lane, "")
			result = b.impl.CreateInsertElement(result, value, llvm.ConstInt(b.Prog.tyInt32(), uint64(base+i), false), "")
		}
	}
	return Expr{result, x.Type}
}

func (b Builder) simdFMA(result Type, x, y, z Expr) Expr {
	target := b.Prog.Target()
	if target.GOOS != "windows" || target.GOARCH != "amd64" {
		v := b.impl.CreateIntrinsic(result.ll, llvm.LookupIntrinsicID("llvm.fma"), []llvm.Value{x.impl, y.impl, z.impl}, "")
		return Expr{v, result}
	}
	// LLVM's baseline fma libcall reaches MinGW's non-fused edge cases.
	// Preserve single rounding without requiring the optional FMA instruction.
	scalar := b.Prog.Type(simdLanes(x.RawType()).Elem(), GoDecl)
	name := "SIMDFMA64"
	if x.ll.ElementType().TypeKind() == llvm.FloatTypeKind {
		name = "SIMDFMA32"
	}
	fn := b.Pkg.rtFunc(RuntimeSymbol(name, "SIMDFMA32", "SIMDFMA64"))
	vector := llvm.Undef(result.ll)
	for i := 0; i < result.ll.VectorSize(); i++ {
		lane := llvm.ConstInt(b.Prog.tyInt32(), uint64(i), false)
		extract := func(v Expr) Expr { return Expr{b.impl.CreateExtractElement(v.impl, lane, ""), scalar} }
		value := b.Call(fn, extract(x), extract(y), extract(z))
		vector = b.impl.CreateInsertElement(vector, value.impl, lane, "")
	}
	return Expr{vector, result}
}
