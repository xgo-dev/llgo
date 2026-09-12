package ssa

import (
	"go/types"

	"github.com/xgo-dev/llvm"
)

// UMulOverflow returns the wrapped unsigned product and an overflow flag.
// Both operands must have the same unsigned integer type.
func (b Builder) UMulOverflow(a, c Expr) Expr {
	t := a.Type
	basic, ok := t.RawType().Underlying().(*types.Basic)
	if !ok || basic.Info()&types.IsUnsigned == 0 || !types.Identical(t.RawType(), c.Type.RawType()) {
		panic("umulOverflow(a, b T) (T, bool): operands must have the same unsigned integer type")
	}
	prog := b.Prog
	// LLVM's result has its native {integer, i1} layout, even on targets
	// where Go aggregates use a different alignment (for example 386).
	retTy := prog.ctx.StructType([]llvm.Type{a.impl.Type(), prog.tyInt1()}, false)
	ret := b.impl.CreateIntrinsic(retTy, llvm.LookupIntrinsicID("llvm.umul.with.overflow"), []llvm.Value{a.impl, c.impl}, "")
	resultType := prog.Struct(t, prog.Bool())
	if retTy != resultType.ll {
		// Normalize before the tuple can escape through a function-value
		// wrapper, which returns the complete Go aggregate at once.
		return b.aggregateValue(resultType,
			b.impl.CreateExtractValue(ret, 0, ""),
			b.impl.CreateExtractValue(ret, 1, ""))
	}
	return Expr{ret, resultType}
}
