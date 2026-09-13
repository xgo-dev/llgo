package ssa

import (
	"go/token"
	"go/types"
	"strings"

	"github.com/xgo-dev/llvm"
)

// SIMD computations use natural LLVM vectors. The surrounding Go values keep
// their existing aggregate storage and call representation. LLVM optimization
// can eliminate the lane packing at inlined computation boundaries.
func (b Builder) simd128Vector(x Expr) llvm.Value {
	if b.Prog.Target().GOARCH == "wasm" {
		features := "+simd128"
		for _, attr := range b.Func.impl.GetFunctionAttributes() {
			if attr.IsString() && attr.GetStringKind() == "target-features" {
				features = attr.GetStringValue()
				if !strings.Contains(","+features+",", ",+simd128,") {
					features += ",+simd128"
				}
			}
		}
		b.Func.impl.AddFunctionAttr(b.Prog.ctx.CreateStringAttribute("target-features", features))
	}
	st := x.RawType().Underlying().(*types.Struct)
	lanes := st.Field(1).Type().(*types.Array)
	elem := b.Prog.toType(lanes.Elem())
	vec := llvm.Undef(llvm.VectorType(elem.ll, int(lanes.Len())))
	values := b.impl.CreateExtractValue(x.impl, 1, "")
	for i := 0; i < int(lanes.Len()); i++ {
		lane := b.impl.CreateExtractValue(values, i, "")
		vec = b.impl.CreateInsertElement(vec, lane, llvm.ConstInt(b.Prog.tyInt32(), uint64(i), false), "")
	}
	return vec
}

func (b Builder) simd128Storage(vec llvm.Value, typ Type) Expr {
	lanes := typ.RawType().Underlying().(*types.Struct).Field(1).Type().(*types.Array)
	array := llvm.Undef(b.Prog.toType(lanes).ll)
	for i := 0; i < int(lanes.Len()); i++ {
		lane := b.impl.CreateExtractElement(vec, llvm.ConstInt(b.Prog.tyInt32(), uint64(i), false), "")
		array = b.impl.CreateInsertValue(array, lane, i, "")
	}
	return Expr{b.impl.CreateInsertValue(llvm.ConstNull(typ.ll), array, 1, ""), typ}
}

// SIMD128Binary lowers a numeric vector family using its actual element type.
func (b Builder) SIMD128Binary(op token.Token, x, y Expr) Expr {
	a, c := b.simd128Vector(x), b.simd128Vector(y)
	floating := x.RawType().Underlying().(*types.Struct).Field(1).Type().(*types.Array).Elem().Underlying().(*types.Basic).Info()&types.IsFloat != 0
	var v llvm.Value
	switch op {
	case token.ADD:
		if floating {
			v = b.impl.CreateFAdd(a, c, "")
		} else {
			v = b.impl.CreateAdd(a, c, "")
		}
	case token.SUB:
		if floating {
			v = b.impl.CreateFSub(a, c, "")
		} else {
			v = b.impl.CreateSub(a, c, "")
		}
	case token.AND:
		v = b.impl.CreateAnd(a, c, "")
	case token.OR:
		v = b.impl.CreateOr(a, c, "")
	case token.XOR:
		v = b.impl.CreateXor(a, c, "")
	default:
		panic("invalid SIMD128 binary operation")
	}
	return b.simd128Storage(v, x.Type)
}

func (b Builder) simd128Index(x, index Expr) llvm.Value {
	lanes := x.RawType().Underlying().(*types.Struct).Field(1).Type().(*types.Array).Len()
	// Check even when ordinary slice bounds checks are disabled: an invalid
	// immediate is an intrinsic error and must never become LLVM poison.
	bad := Expr{llvm.CreateICmp(b.impl, llvm.IntUGE, index.impl, llvm.ConstInt(index.ll, uint64(lanes), false)), b.Prog.Bool()}
	blocks := b.Func.MakeBlocks(2)
	b.If(bad, blocks[0], blocks[1])
	b.SetBlockEx(blocks[0], AtEnd, false)
	b.Call(b.Pkg.rtFunc("PanicSIMDImmediate"))
	b.Unreachable()
	b.SetBlockEx(blocks[1], AtEnd, false)
	b.blk.last = blocks[1].last
	return b.impl.CreateZExt(index.impl, b.Prog.tyInt32(), "")
}

// SIMD128GetElem and SIMD128SetElem also support dynamic uint8 indices.
func (b Builder) SIMD128GetElem(x, index Expr) Expr {
	i := b.simd128Index(x, index)
	v := b.impl.CreateExtractElement(b.simd128Vector(x), i, "")
	elem := x.RawType().Underlying().(*types.Struct).Field(1).Type().(*types.Array).Elem()
	return Expr{v, b.Prog.toType(elem)}
}

func (b Builder) SIMD128SetElem(x, index, value Expr) Expr {
	i := b.simd128Index(x, index)
	v := b.impl.CreateInsertElement(b.simd128Vector(x), value.impl, i, "")
	return b.simd128Storage(v, x.Type)
}
