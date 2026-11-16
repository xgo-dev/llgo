package ssa

import (
	"go/types"
	"math"

	"github.com/xgo-dev/llvm"
)

// simdConvert keeps invalid floating conversions defined. LLVM's plain fptosi
// and fptoui would produce poison for NaNs and out-of-range lanes.
func (b Builder) simdConvert(result Type, x Expr) Expr {
	from := simdLanes(x.RawType()).Elem().Underlying().(*types.Basic).Info()
	to := simdLanes(result.RawType()).Elem().Underlying().(*types.Basic).Info()
	// ConvertLo selects the lower input lanes before widening. Other
	// conversions preserve all lanes; the frontend validates the shape.
	input := x.impl
	if result.ll.VectorSize() < x.ll.VectorSize() {
		mask := make([]llvm.Value, result.ll.VectorSize())
		for i := range mask {
			mask[i] = llvm.ConstInt(b.Prog.tyInt32(), uint64(i), false)
		}
		input = b.impl.CreateShuffleVector(input, llvm.Undef(x.ll), llvm.ConstVector(mask, false), "")
	}
	n := input.Type().VectorSize()
	typ := llvm.VectorType(result.ll.ElementType(), n)
	var v llvm.Value
	switch {
	case from&types.IsFloat != 0 && to&types.IsInteger != 0:
		name := "llvm.fptosi.sat"
		unsigned := to&types.IsUnsigned != 0
		if unsigned {
			name = "llvm.fptoui.sat"
		}
		v = b.impl.CreateIntrinsic(typ, llvm.LookupIntrinsicID(name), []llvm.Value{input}, "")
		if b.Prog.Target().GOARCH == "amd64" {
			width := typ.ElementType().IntTypeWidth()
			upper := math.Ldexp(1, width-1)
			lower := -upper
			invalidResult := uint64(1) << (width - 1)
			if unsigned {
				upper, lower, invalidResult = math.Ldexp(1, width), -1, ^uint64(0)
			}
			splat := func(f float64) llvm.Value {
				vals := make([]llvm.Value, n)
				for i := range vals {
					vals[i] = llvm.ConstFloat(x.ll.ElementType(), f)
				}
				return llvm.ConstVector(vals, false)
			}
			above := b.impl.CreateFCmp(llvm.FloatUGE, input, splat(upper), "")
			pred := llvm.FloatOLT
			if unsigned {
				pred = llvm.FloatOLE
			}
			below := b.impl.CreateFCmp(pred, input, splat(lower), "")
			invalid := b.impl.CreateOr(above, below, "")
			v = b.impl.CreateSelect(invalid, simdIntegerConstant(typ, invalidResult), v, "")
		}
	case from&types.IsFloat != 0 && to&types.IsFloat != 0:
		if x.ll.ElementType().TypeKind() == llvm.FloatTypeKind && typ.ElementType().TypeKind() == llvm.DoubleTypeKind {
			v = b.impl.CreateFPExt(input, typ, "")
		} else {
			v = b.impl.CreateFPTrunc(input, typ, "")
		}
	case to&types.IsFloat != 0:
		if from&types.IsUnsigned != 0 {
			v = b.impl.CreateUIToFP(input, typ, "")
		} else {
			v = b.impl.CreateSIToFP(input, typ, "")
		}
	default:
		v = b.impl.CreateBitCast(input, typ, "")
	}
	if n != result.ll.VectorSize() {
		// Converting to a narrower element type increases the result lane count.
		// Preserve the converted lanes and zero-fill the added upper lanes.
		mask := make([]llvm.Value, result.ll.VectorSize())
		for i := range mask {
			index := i
			if i >= n {
				index = n
			}
			mask[i] = llvm.ConstInt(b.Prog.tyInt32(), uint64(index), false)
		}
		v = b.impl.CreateShuffleVector(v, llvm.ConstNull(typ), llvm.ConstVector(mask, false), "")
	}
	return Expr{v, result}
}
