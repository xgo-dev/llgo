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
	n := x.ll.VectorSize()
	typ := llvm.VectorType(result.ll.ElementType(), n)
	var v llvm.Value
	switch {
	case from&types.IsFloat != 0 && to&types.IsInteger != 0:
		name := "llvm.fptosi.sat"
		unsigned := to&types.IsUnsigned != 0
		if unsigned {
			name = "llvm.fptoui.sat"
		}
		v = b.impl.CreateIntrinsic(typ, llvm.LookupIntrinsicID(name), []llvm.Value{x.impl}, "")
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
			above := b.impl.CreateFCmp(llvm.FloatUGE, x.impl, splat(upper), "")
			pred := llvm.FloatOLT
			if unsigned {
				pred = llvm.FloatOLE
			}
			below := b.impl.CreateFCmp(pred, x.impl, splat(lower), "")
			invalid := b.impl.CreateOr(above, below, "")
			v = b.impl.CreateSelect(invalid, simdIntegerConstant(typ, invalidResult), v, "")
		}
	case from&types.IsFloat != 0 && to&types.IsFloat != 0:
		v = b.impl.CreateFPTrunc(x.impl, typ, "")
	case to&types.IsFloat != 0:
		if from&types.IsUnsigned != 0 {
			v = b.impl.CreateUIToFP(x.impl, typ, "")
		} else {
			v = b.impl.CreateSIToFP(x.impl, typ, "")
		}
	default:
		v = b.impl.CreateBitCast(x.impl, typ, "")
	}
	if n != result.ll.VectorSize() {
		// Narrowing conversions leave the unused upper lanes zero on these targets.
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
